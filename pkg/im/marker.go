// Package im provides read-only structs, slices, and maps checked by immutago.
//
// Choose the API for the data:
//   - Shared: read-only fields; callers can store or return pointers.
//   - Scoped: read-only fields; pointers stay local or pass to checked readers.
//   - Owner: publish a new Scoped object while readers finish with the previous one.
//   - FrozenSlice and FrozenMap: freeze a native collection without copying entries.
//     Use Clone for an independent mutable copy.
//
// Prepare a native collection in one local variable. After Freeze, use only the
// frozen value. Run immutago to enforce these rules; go build alone does not.
package im

// Shared marks a struct with read-only fields. Values and pointers to that struct
// can be copied, stored, and returned through the allowed source forms.
// Embed it by value. A named field or pointer embedding does not mark the struct.
// See https://github.com/lucasavila00/immutago/blob/main/docs/shared.md for the complete forms.
type Shared struct{}

// Scoped marks a struct with read-only fields. Values can be copied, stored, and returned.
// Pointers can stay local or pass to direct checked calls. They cannot be stored
// in fields or containers, saved globally, returned, or captured by closures.
// Scoped does not make a value expire at the end of a Go block.
// Use Owner to publish a new Scoped object for other code to read.
// Embed it by value. A named field or pointer embedding does not mark the struct.
// See https://github.com/lucasavila00/immutago/blob/main/docs/scoped.md for the complete forms.
type Scoped struct{}
