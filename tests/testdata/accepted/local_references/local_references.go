package local_references

import "github.com/lucasavila00/immutago/pkg/im"

type Local struct {
	im.LocalRef
	Value int
}
type Snapshot struct {
	im.Shared
	Value Local
}

func read(value *Local) int       { return value.Value }
func copyValue(value Local) Local { return value }
func use() Snapshot {
	pointer := &Local{Value: 1}
	number := read(pointer)
	copied := *pointer
	copied = Local{Value: number + 1}
	_ = copyValue(copied)
	return Snapshot{Value: copied}
}
func values() im.FrozenSlice[Local] {
	builder := []Local{{Value: 1}}
	frozen := im.FreezeSlice(builder)
	return frozen
}

func allocatedLocal() int {
	value := Local{Value: 1}
	pointer := new(value)
	value = Local{Value: 2}
	return read(pointer)
}

func localCopies() int {
	first := &Local{Value: 1}
	copied := first
	first = &Local{Value: 2}
	var zero *Local
	zero = copied
	return read(zero)
}

func nilReference() bool {
	var pointer *Local = nil
	return pointer == nil
}

type LocalSnapshot struct {
	im.LocalRef
	Value Local
}

func localSnapshot() LocalSnapshot {
	return LocalSnapshot{Value: Local{Value: 1}}
}
