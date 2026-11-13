package immutable

import "slices"

const sliceBits = 5
const sliceWidth = 1 << sliceBits
const sliceMask = sliceWidth - 1

// Slice is an immutable sequence. Its zero value is empty.
// T must be immutable, as checked by the analyzer. Values share a tree of
// 32-element blocks; updates copy one path, never the full sequence.
type Slice[T any] struct {
	root   *sliceNode[T]
	tail   []T
	length int
	shift  uint
}

// sliceNode stores either a leaf block or a branch of the persistent vector.
type sliceNode[T any] struct {
	children []*sliceNode[T]
	values   []T
}

// NewSlice copies values into an immutable sequence in O(n) time.
func NewSlice[T any](values ...T) Slice[T] {
	if len(values) == 0 {
		return Slice[T]{}
	}
	tailStart := (len(values) - 1) & ^sliceMask
	s := Slice[T]{tail: slices.Clone(values[tailStart:]), length: len(values)}
	level := make([]*sliceNode[T], 0, tailStart/sliceWidth)
	for start := 0; start < tailStart; start += sliceWidth {
		level = append(level, &sliceNode[T]{values: slices.Clone(values[start : start+sliceWidth])})
	}
	for len(level) > 1 {
		parents := make([]*sliceNode[T], 0, (len(level)+sliceMask)/sliceWidth)
		for start := 0; start < len(level); start += sliceWidth {
			end := min(start+sliceWidth, len(level))
			parents = append(parents, &sliceNode[T]{children: level[start:end:end]})
		}
		level = parents
		s.shift += sliceBits
	}
	if len(level) > 0 {
		s.root = level[0]
	}
	return s
}

// Len returns the number of elements in O(1) time without allocations.
func (s Slice[T]) Len() int { return s.length }

// At returns the element at index in O(log32(n)) time without allocations.
// It panics if index is out of range.
func (s Slice[T]) At(index int) T {
	s.checkIndex(index)
	if index >= s.length-len(s.tail) {
		return s.tail[index&sliceMask]
	}
	node := s.root
	for shift := s.shift; shift > 0; shift -= sliceBits {
		node = node.children[(index>>shift)&sliceMask]
	}
	return node.values[index&sliceMask]
}

// Values returns an independent slice copy in O(n) time.
func (s Slice[T]) Values() []T {
	if s.length == 0 {
		return nil
	}
	values := make([]T, 0, s.length)
	values = appendSliceValues(values, s.root)
	return append(values, s.tail...)
}

// Set returns a value with one element replaced in O(log32(n)) time and space.
// It panics if index is out of range.
func (s Slice[T]) Set(index int, value T) Slice[T] {
	s.checkIndex(index)
	if index >= s.length-len(s.tail) {
		s.tail = slices.Clone(s.tail)
		s.tail[index&sliceMask] = value
	} else {
		s.root = setSliceNode(s.root, s.shift, index, value)
	}
	return s
}

// Append returns a new sequence with values added at the end.
// It copies the tail once per added block and shares unchanged tree branches.
func (s Slice[T]) Append(values ...T) Slice[T] {
	if len(values) > int(^uint(0)>>1)-s.length {
		panic("immutable.Slice: length overflow")
	}
	for len(values) > 0 {
		if len(s.tail) == sliceWidth {
			s = s.pushTail()
		}
		count := min(sliceWidth-len(s.tail), len(values))
		tail := make([]T, len(s.tail)+count)
		copy(tail, s.tail)
		copy(tail[len(s.tail):], values[:count])
		s.tail = tail
		s.length += count
		values = values[count:]
	}
	return s
}

// checkIndex reports invalid indices before any tree access.
func (s Slice[T]) checkIndex(index int) {
	if index < 0 || index >= s.length {
		panic("immutable.Slice: index out of range")
	}
}

// pushTail moves a full immutable tail into the tree by copying its insertion path.
func (s Slice[T]) pushTail() Slice[T] {
	leaf := &sliceNode[T]{values: s.tail}
	index := s.length - sliceWidth
	switch {
	case s.root == nil:
		s.root = leaf
	case uint(index)>>s.shift >= sliceWidth:
		s.root = &sliceNode[T]{children: []*sliceNode[T]{s.root, slicePath(s.shift, leaf)}}
		s.shift += sliceBits
	default:
		s.root = appendSliceNode(s.root, s.shift, index, leaf)
	}
	s.tail = nil
	return s
}

// slicePath builds the missing branch levels above a leaf.
func slicePath[T any](shift uint, leaf *sliceNode[T]) *sliceNode[T] {
	if shift == 0 {
		return leaf
	}
	return &sliceNode[T]{children: []*sliceNode[T]{slicePath(shift-sliceBits, leaf)}}
}

// appendSliceNode copies an insertion path and shares all other children.
func appendSliceNode[T any](node *sliceNode[T], shift uint, index int, leaf *sliceNode[T]) *sliceNode[T] {
	child := (index >> shift) & sliceMask
	children := make([]*sliceNode[T], max(len(node.children), child+1))
	copy(children, node.children)
	if child == len(node.children) {
		children[child] = slicePath(shift-sliceBits, leaf)
	} else {
		children[child] = appendSliceNode(node.children[child], shift-sliceBits, index, leaf)
	}
	return &sliceNode[T]{children: children}
}

// setSliceNode copies a replacement path and the changed leaf block.
func setSliceNode[T any](node *sliceNode[T], shift uint, index int, value T) *sliceNode[T] {
	if shift == 0 {
		values := slices.Clone(node.values)
		values[index&sliceMask] = value
		return &sliceNode[T]{values: values}
	}
	children := slices.Clone(node.children)
	child := (index >> shift) & sliceMask
	children[child] = setSliceNode(children[child], shift-sliceBits, index, value)
	return &sliceNode[T]{children: children}
}

// appendSliceValues copies leaves in index order without exposing shared storage.
func appendSliceValues[T any](values []T, node *sliceNode[T]) []T {
	if node == nil {
		return values
	}
	if node.values != nil {
		return append(values, node.values...)
	}
	for _, child := range node.children {
		values = appendSliceValues(values, child)
	}
	return values
}
