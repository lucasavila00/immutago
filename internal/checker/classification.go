package checker

import "go/types"

// Classifier caches protected storage and callback queries for one analysis pass.
// Its zero value is ready for use. Types must not change after their first query.
// A Classifier must not be used by concurrent passes.
type Classifier struct {
	nodes   map[classificationKey]*classificationNode
	pending []*classificationNode
	ready   []*classificationNode
}

type classificationMode uint8

const (
	storageMode classificationMode = iota
	callableMode
	constraintMode
)

// classificationKey keeps storage, callback, and constraint walks separate.
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

// Contains reports whether t stores a marked value, sealed collection, or process dependency.
func (c *Classifier) Contains(t types.Type) bool { return c.query(t, storageMode) }

// Callable reports whether t contains a callback contract that carries protected storage.
func (c *Classifier) Callable(t types.Type) bool { return c.query(t, callableMode) }

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
	if IsRuntimeType(node.key.typ, UncheckedDependencyType) {
		if node.key.mode != callableMode {
			c.mark(node)
		}
		return
	}
	if IsLibraryType(node.key.typ, "Owner") || IsRuntimeType(node.key.typ, ProcessMarker) || IsUncheckedValue(node.key.typ) {
		return
	}
	switch node.key.mode {
	case constraintMode:
		c.constraintEdges(node)
	case storageMode:
		if parameter, ok := node.key.typ.(*types.TypeParam); ok {
			c.connect(node, parameter.Constraint(), constraintMode)
		} else if IsCollection(node.key.typ) || MarkerCount(node.key.typ) != 0 {
			c.mark(node)
		} else {
			c.storageEdges(node)
		}
	case callableMode:
		c.callableEdges(node)
	}
}

// storageEdges follows stored components and leaves signatures and interfaces opaque.
func (c *Classifier) storageEdges(node *classificationNode) {
	switch storage := node.key.typ.Underlying().(type) {
	case *types.Pointer:
		c.connect(node, storage.Elem(), storageMode)
	case *types.Array:
		c.connect(node, storage.Elem(), storageMode)
	case *types.Slice:
		c.connect(node, storage.Elem(), storageMode)
	case *types.Map:
		c.connect(node, storage.Key(), storageMode)
		c.connect(node, storage.Elem(), storageMode)
	case *types.Chan:
		c.connect(node, storage.Elem(), storageMode)
	case *types.Struct:
		for field := range storage.Fields() {
			c.connect(node, field.Type(), storageMode)
		}
	}
}

// constraintEdges preserves the separate interface and union traversal of type sets.
func (c *Classifier) constraintEdges(node *classificationNode) {
	switch constraint := node.key.typ.Underlying().(type) {
	case *types.Interface:
		for i := 0; i < constraint.NumEmbeddeds(); i++ {
			c.connect(node, constraint.EmbeddedType(i), constraintMode)
		}
	case *types.Union:
		for i := 0; i < constraint.Len(); i++ {
			c.connect(node, constraint.Term(i).Type(), storageMode)
		}
	default:
		c.connect(node, node.key.typ, storageMode)
	}
}

// callableEdges follows nested callback contracts without treating a marked value as a callback.
func (c *Classifier) callableEdges(node *classificationNode) {
	switch storage := node.key.typ.Underlying().(type) {
	case *types.Signature:
		for _, tuple := range []*types.Tuple{storage.Params(), storage.Results()} {
			for item := range tuple.Variables() {
				c.connect(node, item.Type(), storageMode)
				c.connect(node, item.Type(), callableMode)
			}
		}
	case *types.Interface:
		for method := range storage.Methods() {
			c.connect(node, method.Type(), callableMode)
		}
	case *types.Pointer:
		c.connect(node, storage.Elem(), callableMode)
	case *types.Array:
		c.connect(node, storage.Elem(), callableMode)
	case *types.Slice:
		c.connect(node, storage.Elem(), callableMode)
	case *types.Map:
		c.connect(node, storage.Key(), callableMode)
		c.connect(node, storage.Elem(), callableMode)
	case *types.Chan:
		c.connect(node, storage.Elem(), callableMode)
	case *types.Struct:
		for field := range storage.Fields() {
			c.connect(node, field.Type(), callableMode)
		}
	}
}
