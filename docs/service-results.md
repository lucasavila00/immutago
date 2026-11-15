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

## Example: quota results

Examples from the [configuration project](../examples/configuration/README.md):

| Method | Result |
| --- | --- |
| `FileCache.Text() string` | The last file contents. |
| `Limiter.Take() Decision` | An integer type with the request decision. |
| `Limiter.Usage() Usage` | A struct with `im.ByValue` and the current counts. |

```go
type Usage struct {
    im.ByValue
    Used   int
    Limit  int
    Window int
}
```

Callers receive a value they can keep. Returning Usage does not return a pointer
to the limiter's mutable counters.

## Result types — exhaustive

| Type | Example |
| --- | --- |
| Scalar: boolean, integer, floating-point, complex, or string | `bool`, `int`, `float64`, `complex128`, `string` |
| Defined scalar | `type Count int` |
| Service reference | `supervise.Ref[Worker]`; see [service references](service-references.md). |
| Empty ByValue marker value | `im.ByValue` |
| Struct with `im.ByValue` and proved immutable fields | `Usage` |
| Unnamed array whose elements use this type list | `[2]int`, `[2]Usage`, `[2][3]Usage` |
| Alias of an allowed type | `type Response = Usage` |

A service result is a category of return types, not an im marker. A returned
integer is an ordinary Go value: callers can change it. A returned ByValue value
keeps its read-only fields and [pointer rules](by-value.md).

Fields of a ByValue struct follow the [stored-type rules](common-forms.md#markers-and-stored-types).
They can contain frozen collections, Shared values, Shared pointers, and service references.
For example, return a ByValue struct with a frozen map field when callers need
a collection. The frozen map is not an allowed service result on its own.

Start arguments and method parameters follow the [service contract](services.md#service-contract).
The result rule does not prove service isolation or restrict ordinary mutable inputs.

## Return related data together

Return related fields together when they describe one version. The
[model service](../examples/models/fetcher.go) reads one published model for
both its name and prediction:

```go
func (f *Fetcher) Predict(features im.FrozenSlice[float64]) Prediction {
    current := f.current.Load()
    return Prediction{Name: current.Model.Name, Score: model.Predict(current.Model, features)}
}
```

The caller can keep the Prediction after the service publishes a new model.
The single Load keeps the name and score consistent with the same model.

## Rejected examples — non-exhaustive

| Result | Use instead |
| --- | --- |
| `*Usage` | Declare a value result such as `Read() Usage`; return `*p`. |
| `[]int`, `map[string]int`, or a frozen collection | Put a frozen collection in a struct with `im.ByValue` and return that struct. |
| `error`, `any`, or another interface | Return a struct with `im.ByValue` and the information the caller needs. |
| `(int, bool)` | Return a struct with `im.ByValue` and fields for both values. |
| An unmarked struct, Shared struct or pointer, or `im.UncheckedValue[T]` | Return a struct with `im.ByValue` and allowed fields. |
| `type Counts [2]int` | Return `[2]int`, or a struct with `im.ByValue` and an array field. |

ByValue pointers cannot be saved, returned, or captured. See
[ByValue objects](by-value.md) for their use inside a function.
