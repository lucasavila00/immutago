// Package im provides frozen data and local read projections checked by immutago.
//
// Choose the API for the data:
//   - Copied: record fields that are copied with the value.
//   - Frozen: immutable fields that callers can store and share.
//   - Local: fields that follow the checker's local storage and transport rules.
//   - Cell: publish a snapshot and read the current value through a Projection.
//   - Projection: read selected data without a copy of its backing storage.
//   - FrozenSlice and FrozenMap: freeze a native collection without copying entries.
//     Use Clone for an independent mutable copy.
//
// Prepare a native collection in one local variable. After Freeze, use only the
// frozen value. Run immutago to enforce these rules; go build alone does not.
package im

// Copied marks a record with bool, numeric, array, or Copied fields.
// Copying the record copies all its data. Strings and reference fields are rejected.
// Embed the marker by value. Ordinary copies use normal Go mutation and pointer rules.
// A field inside protected storage keeps its owner's write and address rules.
// See https://github.com/lucasavila00/immutago/blob/main/docs/copied.md for the complete forms.
type Copied struct{}

// Frozen marks a struct with immutable fields. Values and pointers to that
// struct can be copied, stored, and returned through the allowed source forms.
// Embed it by value. A named field or pointer embedding does not mark the struct.
// See https://github.com/lucasavila00/immutago/blob/main/docs/frozen-records.md for the complete forms.
type Frozen struct{}

// Local marks a struct whose fields follow the local projection rules.
// The checker restricts storage and transport of the struct and its pointers.
// Embed it by value. A named field or pointer embedding does not mark the struct.
// See https://github.com/lucasavila00/immutago/blob/main/docs/local.md for the complete forms.
type Local struct{}
