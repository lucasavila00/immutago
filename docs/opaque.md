# Opaque values

`im.Opaque[T]` is one of the two [trust markers](rules.md#trust-markers) of
immutago; the other is `proc.UncheckedDependency[T]`, with a different promise. It holds
a pointer to a value that the analyzer does not check. The analyzer does not
prove that the value is immutable: the package that declares `T` is trusted to
never change it after `im.Seal`.

Use it for an internal structure that a package builds once and only reads
afterwards, but that is not written in the checked forms, such as an engine
with its own scratch types. Other packages store and pass the handle, and reach
the value only through the functions of the declaring package.

```go
type baked struct { weights []float64 }

type Model struct {
    im.Shared
    Name   string
    engine im.Opaque[baked]
}

func Load(name string, weights []float64) *Model {
    return &Model{Name: name, engine: im.Seal(&baked{weights: slices.Clone(weights)})}
}

func Predict(m *Model, row int) float64 {
    b := im.Unseal(m.engine)
    return b.weights[row]
}
```

Load copies the input with `slices.Clone`, so later changes by the caller do not
change the sealed object. Seal itself does not copy the data.

The [engine](../tests/testdata/accepted/opaque/engine.go) and
[user](../tests/testdata/accepted/opaqueuse/use.go) examples show both sides.

## Allowed forms

These allowed forms for `im.Seal` and `im.Unseal` are complete. Other uses of
these functions are rejected. The handle follows ordinary Go rules; a marked
object that stores it still follows the [common forms](common-forms.md).

| Allowed form | Example |
| --- | --- |
| Use as an allowed stored type in a marked struct, a frozen collection, or ordinary Go | `type Model struct { im.Shared; engine im.Opaque[baked] }` |
| Copy, store, pass, or return the handle in any package | `other := m.engine` |
| Declare a zero handle; Unseal returns nil | `var engine im.Opaque[baked]` |
| In the package that declares the named type `T`, wrap a pointer | `engine := im.Seal(&baked{})` |
| In the package that declares the named type `T`, read the pointer | `b := im.Unseal(m.engine)` |
| In that package, use either function as a function value | `read := im.Unseal[baked]`; `b := read(engine)` |

Aliases resolve to the original type and package. An instance of a named type,
such as `baked[int]`, belongs to the package that declares `baked`. Parentheses
preserve the forms above. `Opaque` has no methods; an interface call or a promoted
method cannot expose its pointer.

What the analyzer proves:

- The handle is an allowed stored type, so a marked object that holds it
  follows every other rule: its fields cannot be written and its handle cannot
  be replaced.
- Only the package that declares `T` can wrap or read the value.

What is trusted and not proven:

- The declaring package does not change the value after `im.Seal`, and its
  functions do not give out a way to change it. The functions of the declaring
  package are the read-only proxy of the value; review them as trusted code.

## Rejected examples

These examples are not exhaustive.

| Rejected example | Allowed replacement |
| --- | --- |
| `im.Seal(&engine.Table{})` in a package that does not declare `Table` | Call a function of the package that declares `Table`. |
| `im.Unseal(table)` or `read := im.Unseal[engine.Table]` in another package | Call a function of the declaring package. |
| `func Wrap[T any](v *T) im.Opaque[T] { return im.Seal(v) }` | Wrap a concrete named type that this package declares. |
| `im.Seal(&rows)` with `rows []int` | Declare a named type, such as `type rows []int`. |
| `m.engine = im.Seal(&baked{})` on a marked value | Construct a new marked value. |
