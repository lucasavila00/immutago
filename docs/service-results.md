# Service results

A service is a struct that embeds `supervise.Service[A]`. Each exported service
method can return no result or one **service result** from the types below.
This includes promoted methods on named or anonymous service structs.
`Init`, `Run`, and methods supplied by the supervise runtime are exempt.

This rule applies to service methods. Ordinary functions and methods use the
[im source forms](common-forms.md) when they work with protected data.

## Method signatures — exhaustive

| Form | Example |
| --- | --- |
| No result | `func (c *Counter) Pause() { c.paused.Store(true) }` |
| One allowed result | `func (l *Limiter) Usage() Usage` |

## Result types — exhaustive

Unlisted result types are rejected. This list applies to the whole result and
to elements of result arrays. Struct fields use the separate list below.

| Type | Example |
| --- | --- |
| Boolean, integer, floating-point, or complex scalar | `bool`, `int`, `float64`, `complex128` |
| Defined type with one of those scalar kinds | `type Count int` |
| Service reference | `supervise.Ref[Worker]`; see [service references](service-references.md). |
| Empty ByValue marker value | `im.ByValue` |
| Struct with `im.ByValue`, immutable fields, and no shared data references | `Usage` below |
| Local frozen view with allowed type arguments | `im.FrozenByValueString`, `im.FrozenByValueSlice[int]`, `im.FrozenByValueMap[im.FrozenString, int]` |
| Unnamed array whose elements use this result list | `[2]int`, `[2]Usage`, `[2]im.FrozenByValueString` |
| Alias of an allowed type | `type Response = Usage` |

### Fields inside result records — exhaustive

These fields must also pass the [stored-type proof](common-forms.md#markers-and-stored-types).
Apply this list recursively to every record field and array element. Unlisted
field types are rejected.

| Field type | Example |
| --- | --- |
| Boolean or numeric scalar, including a defined type | `int`, `type Count int` |
| Empty marker | `im.Shared`, `im.ByValue` |
| Marked struct with fields from this list | `type Point struct { im.Shared; X int }` |
| Unnamed array of fields from this list | `[2]Point` |
| Local frozen view with its allowed type arguments | `im.FrozenByValueString`, `im.FrozenByValueSlice[int]` |
| Service reference | `supervise.Ref[Worker]` |
| Alias of a listed field type | `type Position = Point` |

For example, `type Reply struct { im.ByValue; P Point }` is an allowed result.
`Point` and `[2]Point` alone are not allowed results because their outer type is
Shared. Shared records cannot contain local views; use ByValue for those records.

Fields cannot contain native strings, shared frozen handles, pointers, or
`im.UncheckedValue` at any depth. Local frozen views use their separate
[type argument list](local-views.md#type-arguments--exhaustive). Service references
remain a separate allowed reference category; callers can keep them.

A ByValue record that contains a view stays local as a whole. The caller may
read, copy locally, pass directly, or return it, but cannot save the record in
persistent storage. Reading a field keeps its original type and rules. See
[local transport](local-views.md#transport--exhaustive).

A service result is a return type category. It is not a marker or a promise
of service isolation. Start arguments, mutable method inputs, and service
implementation code follow the [service contract](services.md#service-contract).

## Example: quota results

Examples from the [configuration project](../examples/configuration/README.md):

| Method | Result |
| --- | --- |
| `FileCache.Text() im.FrozenByValueString` | A local view of the last file contents. |
| `Limiter.Take() Decision` | An integer type with the request decision. |
| `Limiter.Usage() Usage` | A ByValue struct with the current counts. |

```go
type Usage struct {
    im.ByValue
    Used   int
    Limit  int
    Window int
}
```

Callers can keep the scalar Usage value. It has no references to the limiter's
mutable counters. To keep independent file text, call `cache.Current().Text().Clone()`.

## Return related data together

The [model service](../examples/models/fetcher.go) reads one published model
for both its name and prediction:

```go
type Prediction struct {
    im.ByValue
    Name  im.FrozenByValueString
    Score float64
}

func (f *Fetcher) Predict(features im.FrozenSlice[float64]) Prediction {
    current := f.current.Load()
    return Prediction{Name: current.Model.Name.ByValue(), Score: model.Predict(current.Model, features)}
}
```

The name view and score describe the same model version. Prediction stays local
because Name is a view. A caller can save `prediction.Name.Clone()` and
`prediction.Score` as independent data.

Internal ByValue state can still contain shared frozen data or Shared pointers
and can be published through [Owner](owner.md). Such a state type is not a
service result. Build a result with scalar fields and local views instead.

## Rejected examples — non-exhaustive

| Result | Use instead |
| --- | --- |
| `string`, a defined string type, or `im.FrozenString` | Return `im.FrozenByValueString`. |
| `*Usage` | Declare a value result such as `Read() Usage`; return `*p`. |
| `[]int` or `im.FrozenSlice[int]` | Return `im.FrozenByValueSlice[int]`. |
| `map[string]int` or `im.FrozenMap[im.FrozenString, int]` | Return `im.FrozenByValueMap[im.FrozenString, int]`. |
| A ByValue struct with a shared frozen field or Shared pointer | Return a separate ByValue result with local views or scalar copies. |
| `error`, `any`, or another interface | Return a ByValue struct with the information the caller needs. |
| `(int, bool)` | Return a ByValue struct with fields for both values. |
| An unmarked struct, Shared struct, or `im.UncheckedValue[T]` | Return a ByValue struct with allowed fields. |
| `type Counts [2]int` | Return `[2]int`, or a ByValue struct with an array field. |

ByValue pointers cannot be saved, returned, or captured. See
[ByValue objects](by-value.md) for their use inside a function.
