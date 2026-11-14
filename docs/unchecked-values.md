# Unchecked immutable values

`im.UncheckedValue[T]` holds a `*T`: Trust stores the pointer, and Unwrap returns it.
Use it when your package supplies immutable data that does not fit the checked forms:

- Your package promises not to change the data; the checker does not prove this.
- Only the package that declares `T` can wrap it with `im.Trust` or read it with `im.Unwrap`.
- Other packages can keep the handle and call your functions to read the data.

```go
type baked struct { weights []float64 }

type Model struct {
    im.Shared
    Name   string
    engine im.UncheckedValue[baked]
}

func Load(name string, weights []float64) *Model {
    return &Model{Name: name, engine: im.Trust(&baked{weights: slices.Clone(weights)})}
}

func Predict(m *Model, features im.FrozenSlice[float64]) float64 {
    b := im.Unwrap(m.engine)
    if features.Len() != len(b.weights) {
        panic("feature count does not match model")
    }
    var score float64
    for i, weight := range b.weights {
        score += weight * features.At(i)
    }
    return score
}
```

Load copies the input with `slices.Clone`, so later changes by the caller do not
change the trusted object. `im.Trust` itself does not copy or freeze the data.

The [model package](../examples/models/model/model.go) owns the trusted pointer.
The [HTTP service](../examples/models/fetcher.go) uses its public prediction API.
Run the [model project](../examples/models/README.md) to load models and make
predictions while their data changes.

## Allowed forms — exhaustive

The handle follows ordinary Go rules; a marked object that stores it still
follows the [common forms](common-forms.md).

| Allowed form | Example |
| --- | --- |
| Use as an allowed stored type in a marked struct, a frozen collection, or ordinary Go | `type Model struct { im.Shared; engine im.UncheckedValue[baked] }` |
| Copy, store, pass, or return the handle in any package | `other := m.engine` |
| Declare a zero handle; Unwrap returns nil | `var engine im.UncheckedValue[baked]` |
| In the package that declares the named type `T`, wrap a pointer | `engine := im.Trust(&baked{})` |
| In the package that declares the named type `T`, read the pointer | `b := im.Unwrap(m.engine)` |
| In that package, use either function as a function value | `read := im.Unwrap[baked]`; `b := read(engine)` |

Aliases resolve to the original type and package. An instance of a named type,
such as `baked[int]`, belongs to the package that declares `baked`. Parentheses
preserve the forms above. `UncheckedValue` has no methods; an interface call or
a promoted method cannot expose its pointer.

| Responsibility | Who checks it |
| --- | --- |
| A marked object's handle field cannot be replaced | Analyzer |
| Only the package that declares `T` can call Trust or Unwrap | Analyzer |
| The data never changes after Trust, and no function exposes a way to change it | Declaring package; review its code |

## Rejected examples — non-exhaustive

| Rejected example | Allowed replacement |
| --- | --- |
| `im.Trust(&engine.Table{})` in a package that does not declare `Table` | Call a function of the package that declares `Table`. |
| `im.Unwrap(table)` or `read := im.Unwrap[engine.Table]` in another package | Call a function of the declaring package. |
| `func Wrap[T any](v *T) im.UncheckedValue[T] { return im.Trust(v) }` | Wrap a concrete named type that this package declares. |
| `im.Trust(&rows)` with `rows []int` | Declare a named type, such as `type rows []int`. |
| `m.engine = im.Trust(&baked{})` on a marked value | Construct a new marked value. |
