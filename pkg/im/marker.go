// Package im provides read-only structs, slices, and maps checked by immutago.
//
// Embed Shared in a struct when callers can save or return its pointer. Embed
// Scoped when callers must keep the pointer local or pass it to functions that
// do not save it. Both markers require read-only fields, including nested data.
// Owner lets readers load the current Scoped object while a publisher replaces it.
//
// Build slices and maps with normal Go operations, then call FreezeSlice or
// FreezeMap. Freeze reuses their storage without copying entries. Read through
// the frozen value after that call. Clone returns a separate slice or map to edit.
//
// Run immutago on the packages that use these types. Go build alone does not
// enforce these rules. See the guides and complete supported operations at
// https://github.com/lucasavila00/immutago/blob/main/docs/rules.md.
package im

// Shared marks a struct with read-only fields. Its pointers can be saved or returned.
// Embed it by value. A named field or pointer embedding does not mark the struct.
// See https://github.com/lucasavila00/immutago/blob/main/docs/shared.md for the complete forms.
type Shared struct{}

// Scoped marks a struct with read-only fields. Its pointers cannot be saved or returned.
// Use Owner to publish a new Scoped object for other code to read.
// Embed it by value. A named field or pointer embedding does not mark the struct.
// See https://github.com/lucasavila00/immutago/blob/main/docs/scoped.md for the complete forms.
type Scoped struct{}
