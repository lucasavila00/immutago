package checker

import "go/types"

// Classifier caches protected storage, callback, and local storage queries for one analysis pass.
// Its zero value is ready for use. Types must not change after their first query.
// A Classifier must not be used by concurrent passes.
type Classifier struct {
	nodes   map[classificationKey]*classificationNode
	pending []*classificationNode
	ready   []*classificationNode
}

type classificationMode uint8

const (
	sourceMode classificationMode = iota
	ownershipMode
	transportMode
	callableMode
	localMode
)

// classificationKey keeps each traversal mode separate.
type classificationKey struct {
	typ  types.Type
	mode classificationMode
}

// classificationNode records reverse edges and a proved positive classification.
type classificationNode struct {
	key   classificationKey
	users []*classificationNode
	value bool
}

// Contains reports values that need im source checks, including their holders and containers.
// Frozen pointer handles and Copied records use ordinary Go storage rules.
func (c *Classifier) Contains(t types.Type) bool { return c.query(t, sourceMode) }

// Owns reports protected object storage, including its holders and containers.
// Frozen handles can occupy ordinary mutable slots without changing their targets.
func (c *Classifier) Owns(t types.Type) bool { return c.query(t, ownershipMode) }

// Transport reports values that need restricted calls, bindings, and captures.
// Frozen values and pointers can persist. Local values and protected native container uses stay restricted.
func (c *Classifier) Transport(t types.Type) bool { return c.query(t, transportMode) }

// Callable reports whether t contains a callback contract that carries protected storage.
func (c *Classifier) Callable(t types.Type) bool { return c.query(t, callableMode) }

// Locals reports local projections or records in storage, including through arrays and pointers.
func (c *Classifier) Locals(t types.Type) bool { return c.query(t, localMode) }

// query completes all reachable edges before propagating positive results through cycles.
func (c *Classifier) query(t types.Type, mode classificationMode) bool {
	if t == nil {
		return false
	}
	key := classificationKey{types.Unalias(t), mode}
	if node, found := c.nodes[key]; found {
		return node.value
	}
	if c.nodes == nil {
		c.nodes = make(map[classificationKey]*classificationNode)
	}
	root := c.node(key)
	for index := 0; index < len(c.pending); index++ {
		c.expand(c.pending[index])
	}
	for index := 0; index < len(c.ready); index++ {
		for _, owner := range c.ready[index].users {
			c.mark(owner)
		}
	}
	c.pending = c.pending[:0]
	c.ready = c.ready[:0]
	return root.value
}

// node registers a type before its edges are expanded, so recursive types terminate.
func (c *Classifier) node(key classificationKey) *classificationNode {
	key.typ = types.Unalias(key.typ)
	if existing, found := c.nodes[key]; found {
		return existing
	}
	node := &classificationNode{key: key}
	c.nodes[key] = node
	c.pending = append(c.pending, node)
	return node
}

// connect adds an edge without treating an unfinished negative result as final.
func (c *Classifier) connect(owner *classificationNode, t types.Type, mode classificationMode) {
	if t == nil {
		return
	}
	child := c.node(classificationKey{t, mode})
	child.users = append(child.users, owner)
	if child.value {
		c.mark(owner)
	}
}

// mark queues a positive classification only once.
func (c *Classifier) mark(node *classificationNode) {
	if !node.value {
		node.value = true
		c.ready = append(c.ready, node)
	}
}

// expand matches the outgoing edges for one classification mode.
func (c *Classifier) expand(node *classificationNode) {
	if node.key.mode == localMode {
		if IsProjection(node.key.typ) || Local(node.key.typ) || IsLibraryType(node.key.typ, "Local") {
			c.mark(node)
		} else {
			c.storageEdges(node)
		}
		return
	}
	if IsHandle(node.key.typ) {
		if node.key.mode == sourceMode {
			c.mark(node)
		}
		return
	}
	if IsLibraryType(node.key.typ, "Cell") || IsRuntimeType(node.key.typ, ServiceMarker) ||
		IsUncheckedValue(node.key.typ) {
		return
	}
	switch node.key.mode {
	case sourceMode, ownershipMode, transportMode:
		c.storedValue(node)
	case callableMode:
		c.callableEdges(node)
	}
}

// storedValue identifies types whose storage needs protected operation checks.
func (c *Classifier) storedValue(node *classificationNode) {
	if FrozenPointer(node.key.typ) {
		return
	}
	if parameter, ok := node.key.typ.(*types.TypeParam); ok {
		c.connect(node, parameter.Constraint(), node.key.mode)
	} else if IsProjection(node.key.typ) || IsLibraryType(node.key.typ, "Local") || Local(node.key.typ) {
		c.mark(node)
	} else if Frozen(node.key.typ) {
		if node.key.mode != transportMode {
			c.mark(node)
		}
	} else if IsCollection(node.key.typ) || FrozenInterface(node.key.typ) {
		if node.key.mode == sourceMode {
			c.mark(node)
		}
	} else {
		c.storageEdges(node)
	}
}

// storageEdges follows stored components and recognizes sealed interfaces as protected storage.
func (c *Classifier) storageEdges(node *classificationNode) {
	switch storage := node.key.typ.Underlying().(type) {
	case *types.Signature:
		if node.key.mode == sourceMode {
			c.callableEdges(node)
		}
	case *types.Interface:
		if node.key.mode == localMode {
			return
		}
		for i := 0; i < storage.NumEmbeddeds(); i++ {
			c.connect(node, storage.EmbeddedType(i), node.key.mode)
		}
	case *types.Union:
		for i := 0; i < storage.Len(); i++ {
			c.connect(node, storage.Term(i).Type(), node.key.mode)
		}
	case *types.Pointer:
		c.referenceEdge(node, storage.Elem())
	case *types.Array:
		c.connect(node, storage.Elem(), node.key.mode)
	case *types.Slice:
		c.referenceEdge(node, storage.Elem())
	case *types.Map:
		c.referenceEdge(node, storage.Key())
		c.referenceEdge(node, storage.Elem())
	case *types.Chan:
		c.referenceEdge(node, storage.Elem())
	case *types.Struct:
		for field := range storage.Fields() {
			c.connect(node, field.Type(), node.key.mode)
		}
	case *types.Tuple:
		for item := range storage.Variables() {
			c.connect(node, item.Type(), node.key.mode)
		}
	}
}

// FrozenPointer identifies a pointer to a Frozen record by its resolved target type.
func FrozenPointer(t types.Type) bool {
	if t == nil {
		return false
	}
	pointer, ok := t.Underlying().(*types.Pointer)
	return ok && Frozen(pointer.Elem())
}

// referenceEdge keeps Local pointers and protected native container transport restricted.
func (c *Classifier) referenceEdge(node *classificationNode, element types.Type) {
	mode := node.key.mode
	if mode == transportMode {
		mode = ownershipMode
	}
	c.connect(node, element, mode)
}

// callableEdges follows stored components and callable contracts in the current mode.
func (c *Classifier) callableEdges(node *classificationNode) {
	switch storage := node.key.typ.Underlying().(type) {
	case *types.Signature:
		for _, tuple := range []*types.Tuple{storage.Params(), storage.Results()} {
			for item := range tuple.Variables() {
				if node.key.mode == callableMode {
					c.connect(node, item.Type(), transportMode)
				}
				c.connect(node, item.Type(), node.key.mode)
			}
		}
	case *types.Interface:
		if node.key.mode == callableMode && FrozenInterface(node.key.typ) {
			return
		}
		for method := range storage.Methods() {
			c.connect(node, method.Type(), node.key.mode)
		}
	case *types.Pointer:
		c.connect(node, storage.Elem(), node.key.mode)
	case *types.Array:
		c.connect(node, storage.Elem(), node.key.mode)
	case *types.Slice:
		c.connect(node, storage.Elem(), node.key.mode)
	case *types.Map:
		c.connect(node, storage.Key(), node.key.mode)
		c.connect(node, storage.Elem(), node.key.mode)
	case *types.Chan:
		c.connect(node, storage.Elem(), node.key.mode)
	case *types.Struct:
		for field := range storage.Fields() {
			c.connect(node, field.Type(), node.key.mode)
		}
	}
}
