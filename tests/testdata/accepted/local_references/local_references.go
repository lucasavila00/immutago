package local_references

import "github.com/lucasavila00/immutago/pkg/im"

type Local struct {
	im.Local
	Value int
}
type LocalRecord struct {
	im.Local
	Value Local
}

func read(value *Local) int       { return value.Value }
func copyValue(value Local) Local { return value }
func use() LocalRecord {
	pointer := &Local{Value: 1}
	number := read(pointer)
	copied := *pointer
	copied = Local{Value: number + 1}
	_ = copyValue(copied)
	return LocalRecord{Value: copied}
}
func values() [1]Local {
	return [1]Local{{Value: 1}}
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
	im.Local
	Value Local
}

func localSnapshot() LocalSnapshot {
	return LocalSnapshot{Value: Local{Value: 1}}
}
