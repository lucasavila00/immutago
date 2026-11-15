package protected

import (
	r "reflect"
	u "unsafe"

	"github.com/lucasavila00/immutago/pkg/im"
)

type Record struct {
	im.Frozen
	Name string
}

type Cache struct {
	current *Record
}

type UnsafeField struct {
	im.Frozen
	Pointer u.Pointer
}

// Pointer cannot erase the protected pointer type.
func Pointer(record *Record) { _ = u.Pointer(record) }

// Reconstruct cannot create a protected pointer by conversion.
func Reconstruct(value u.Pointer) *Record { return (*Record)(value) }

// Reflect cannot box a protected pointer.
func Reflect(record *Record) { r.ValueOf(record).Elem().FieldByName("Name").SetString("changed") }

// Type cannot box a protected value for reflection metadata.
func Type(record Record) { _ = r.TypeOf(record) }

// GenericType cannot hide a protected type in a generic reflection call.
func GenericType() { _ = r.TypeFor[Record]() }

// Slice cannot expose a native container of protected elements.
func Slice(values []Record) { _ = r.ValueOf(values) }

// Frozen cannot expose a frozen wrapper through reflection.
func Frozen(values im.FrozenSlice[int]) { _ = r.ValueOf(values) }
