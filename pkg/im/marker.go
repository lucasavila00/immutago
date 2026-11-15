// Package im provides read-only structs, slices, maps, and strings checked by immutago.
//
// Choose the API for the data:
//   - Shared: read-only fields; callers can store or return pointers.
//   - ByValue: keep read-only value copies; use pointers through direct checked calls.
//   - Owner: publish a new ByValue object while readers finish with the previous one.
//   - FrozenSlice, FrozenMap, and FrozenString: share read-only storage without a copy.
//   - FrozenByValueSlice, FrozenByValueMap, and FrozenByValueString: read storage locally.
//     Use Clone for independent native storage; nested local views require explicit copies.
//
// Prepare a native collection in one local variable. After Freeze, use only the
// frozen value. Run immutago to enforce these rules; go build alone does not.
package im

// Shared marks a struct with read-only fields. Values and pointers to that struct
// can be copied, stored, and returned through the allowed source forms.
// Embed it by value. A named field or pointer embedding does not mark the struct.
// See https://github.com/lucasavila00/immutago/blob/main/docs/shared.md for the complete forms.
type Shared struct{}

// ByValue marks a struct with read-only fields. Values can be copied and returned.
// A value with local frozen views must follow their local storage rules.
// Other values can be stored through the allowed source forms.
// Pointers can stay local or pass to direct checked calls. They cannot be stored
// in fields or containers, saved globally, returned, or captured by closures.
// Use Owner to publish a ByValue object that contains no local frozen views.
// Embed it by value. A named field or pointer embedding does not mark the struct.
// See https://github.com/lucasavila00/immutago/blob/main/docs/by-value.md for the complete forms.
type ByValue struct{}
