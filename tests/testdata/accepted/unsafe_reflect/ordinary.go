package ordinary

import (
	"reflect"
	"unsafe"

	"github.com/lucasavila00/immutago/pkg/im"
)

type Record struct {
	im.Shared
	Name im.FrozenString
}

// First reads a string through an ordinary unsafe helper.
func First(s string) byte {
	data := unsafe.Slice(unsafe.StringData(s), len(s))
	if len(data) == 0 {
		return 0
	}
	return data[0]
}

// NameFirst passes an independent string copy to an ordinary helper.
func NameFirst(record *Record) byte { return First(record.Name.Clone()) }

// NameData exposes the storage of an independent string copy.
func NameData(record *Record) *byte { return unsafe.StringData(record.Name.Clone()) }

// CopiedNameData uses an ordinary local string.
func CopiedNameData(record *Record) *byte {
	name := record.Name.Clone()
	return unsafe.StringData(name)
}

// Set changes an ordinary integer through reflection.
func Set(value *int) { reflect.ValueOf(value).Elem().SetInt(1) }

// Pointer exposes an ordinary pointer through an unsafe conversion.
func Pointer(value *int) unsafe.Pointer { return unsafe.Pointer(value) }

// Type returns metadata for a type that has no protected storage.
func Type[T any]() reflect.Type { return reflect.TypeFor[T]() }
