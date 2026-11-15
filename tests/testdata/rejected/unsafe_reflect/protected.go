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

type Local struct {
	im.Local
	N int
}

type Cache struct {
	current *Record
}

type UnsafeField struct {
	im.Frozen
	Pointer u.Pointer
}

// Pointer uses an ordinary Go pointer conversion.
func Pointer(record *Record) { _ = u.Pointer(record) }

// Reconstruct uses an ordinary Go pointer conversion.
func Reconstruct(value u.Pointer) *Record { return (*Record)(value) }

// Reflect cannot box a Local pointer.
func Reflect(record *Local) { r.ValueOf(record).Elem().FieldByName("N").SetInt(1) }

// Type cannot box a Local value for reflection metadata.
func Type(record Local) { _ = r.TypeOf(record) }

// GenericType cannot hide a Local type in a generic reflection call.
func GenericType() { _ = r.TypeFor[Local]() }

// Slice cannot expose a native container of protected elements.
func Slice(values []Record) { _ = r.ValueOf(values) }

// Frozen can pass a persistent wrapper to an ordinary reflection call.
func Frozen(values im.FrozenSlice[int]) { _ = r.ValueOf(values) }
