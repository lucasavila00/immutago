# Local frozen views

`im.FrozenByValueSlice`, `im.FrozenByValueMap`, and `im.FrozenByValueString`
provide local read access to frozen storage. They share the storage; they do
not copy entries or bytes. A view can be passed to a direct checked call or
returned to its caller. It cannot be saved in persistent protected storage.

Choose a [shared frozen value](frozen.md) when callers must store the handle.
Choose a local view for a [service result](service-results.md) or a reader that
must not save the handle. Clone data to keep an independent copy.

The rules do not set a time limit on a reader. A local variable can keep a view
alive while its function runs. This is a source restriction, not a runtime loan
or a proof of service isolation.

## Supported operations — exhaustive

Unlisted protected operations are rejected. All methods must use direct calls
with an allowed [frozen receiver](frozen.md#frozen-method-receivers--exhaustive).

| Operation | `FrozenByValueSlice[T]` | `FrozenByValueMap[K, V]` | `FrozenByValueString` |
| --- | --- | --- | --- |
| Zero value | `var view im.FrozenByValueSlice[int]` | `var view im.FrozenByValueMap[int, int]` | `var view im.FrozenByValueString` |
| Freeze native input | `view := im.FreezeByValueSlice(values)` | `view := im.FreezeByValueMap(values)` | `view := im.FreezeByValueString(text)` |
| View of the matching shared type | `view := frozen.ByValue()` | `view := frozen.ByValue()` | `view := frozen.ByValue()` |
| Length | `view.Len()` | `view.Len()` | `view.Len()` |
| Read one value | `value := view.At(i)` | `value, found := view.Get(key)` | `value := view.At(i)` |
| Direct range | `for i, value := range view.All() { ... }` | Same form; unavailable for `FrozenString` keys. | Use byte indices and `At`. |
| Independent native copy | `values := view.Clone()` | `values := view.Clone()` | `text := view.Clone()` |

- Slice and map Freeze use all the [builder forms](frozen.md#build-before-freezing)
  and consume the builder. String Freeze leaves the input usable.
- `.ByValue()` preserves the shared collection's type arguments. They must pass
  the type list below. It does not convert each element to a view.
- Slice and map Clone are allowed only when their type arguments contain no
  local views. They copy slots. Map Clone also copies `FrozenString` key bytes.
- Every local `All()` call must be the operand of a direct range loop, even
  when it yields only integers. Its iterator holds the view.
- Empty literals, aliases, and parentheses preserve the listed forms.

## Type arguments — exhaustive

Each key, element, value, and nested field must use this list. Map keys must
also be comparable in Go.

| Allowed type | Example |
| --- | --- |
| Boolean, integer, floating-point, or complex scalar, including defined types | `bool`, `int`, `float64`, `type Count int` |
| Empty marker value | `im.Shared`, `im.ByValue` |
| Unnamed array of allowed elements | `[2]int`, `[2]im.FrozenByValueString` |
| Marked struct with allowed fields | `type Point struct { im.Shared; X int }` |
| Local frozen view with allowed type arguments | `im.FrozenByValueString`, `im.FrozenByValueSlice[int]` |
| Service reference | `supervise.Ref[Worker]` |
| Alias of an allowed type | `type Count = int` |
| Exact `im.FrozenString`, for a map key only | `im.FrozenByValueMap[im.FrozenString, int]` |

A struct with a local view field must use `im.ByValue`. Shared pointers,
shared frozen values, native strings, and `im.UncheckedValue` are not allowed
in yielded elements or values. The map-key exception permits lookup without
returning the stored key; it does not permit `All()`.

| Key family | Get | Direct All | Clone |
| --- | --- | --- | --- |
| Copied data, such as `int` | Allowed | Allowed | Allowed if values contain no local views. |
| Exact `im.FrozenString` | Allowed | Rejected | Copies keys and slots if values contain no local views. |
| `im.FrozenByValueString` or another allowed key containing views | Allowed | Allowed; yielded keys stay local. | Rejected; copy each key explicitly. |

Service references retain their separate [handle rules](service-references.md).
A view can yield a reference that callers keep. It does not expose the service's
mutable state as immutable data.

## Example: look up a text key

```go
func score(scores im.FrozenByValueMap[im.FrozenString, int], name string) int {
    value, _ := scores.Get(im.FreezeString(name))
    return value
}

func readScores() {
    scores := im.FreezeMap(map[im.FrozenString]int{im.FreezeString("alice"): 10})
    view := scores.ByValue()
    println(score(view, "alice"))
    saved := view.Clone()
    for key, value := range saved {
        println(key.Clone(), value)
    }
}
```

Clone makes a new map and copies the stored text key bytes. `Get` returns only
the integer and does no copy.

## Transport — exhaustive

Use these forms with the [common statement forms](common-forms.md#statements-and-calls).
The same rules apply to a view and to every ByValue struct or unnamed array
that contains one, at any depth.

| Allowed form | Example |
| --- | --- |
| Declare, copy, replace, or discard a local value | `view := frozen.ByValue()`, `other := view`, `view = other`, `_ = view` |
| Put a view in a ByValue field or unnamed array | `Result{Text: view}`, `[1]im.FrozenByValueString{view}` |
| Read a field, array element, or frozen entry | `text := result.Text`, `text := views.At(0)` |
| Put views in a local builder for the local frozen family | `values := []im.FrozenByValueString{view}`; `frozen := im.FreezeByValueSlice(values)` |
| Pass to a direct concrete function or method | `read(view)` with `func read(v im.FrozenByValueString)` |
| Return through a concrete result | `return view` with `func text() im.FrozenByValueString` |

Reading a view from a field or entry keeps its view type and restrictions.
Copying a containing record does not remove them. A return transfers the same
restricted value to the caller; it does not permit persistent storage.

## Example: a list of text views

`FrozenSlice[FrozenString].ByValue()` is rejected because its `At` method would
return shareable strings. Build the required element type explicitly:

```go
func namesView(names im.FrozenSlice[im.FrozenString]) im.FrozenByValueSlice[im.FrozenByValueString] {
    values := make([]im.FrozenByValueString, 0, names.Len())
    for _, name := range names.All() {
        values = append(values, name.ByValue())
    }
    view := im.FreezeByValueSlice(values)
    return view
}
```

This builder copies O(n) view headers. It does not copy string bytes. Freeze
adds no copy. Clone of the resulting slice is rejected because its elements
are local views. To retain the text, clone each element into a new string value.

## Rejected examples — non-exhaustive

| Rejected form | Allowed replacement |
| --- | --- |
| `var saved im.FrozenByValueString` at package scope | Save `view.Clone()` as a native string. |
| `type Saved struct { im.Shared; Text im.FrozenByValueString }` | Use `im.ByValue` for a local result, or store independent data. |
| Store a containing ByValue record in a package variable | Copy the fields you need into independent data first. |
| `im.Owner[Result]` where Result contains a view | Store shared frozen data in a separate ByValue state type; return a view from that state. |
| A view in service start arguments | Supply a shared frozen value or an independent native copy. |
| `im.FrozenSlice[im.FrozenByValueString]` | Use `im.FrozenByValueSlice[im.FrozenByValueString]` locally. |
| `im.FrozenByValueSlice[im.FrozenString]` or `im.FrozenByValueSlice[*Point]` | Build view elements explicitly, or copy scalar fields into a marked value. |
| `view.All()` saved in an iterator variable | Use `for _, value := range view.All() { ... }`. |
| `All()` on a local map with `FrozenString` keys | Use `Get`, or Clone and range over the independent map. |
| `Clone()` on a local collection that contains views | Clone each nested value explicitly and build the desired result. |
| A view in an interface, closure, callback, `go`, `defer`, or user generic | Read scalar data or Clone first; transport that result. |

See [operation costs](performance.md) for the cost of each form.
