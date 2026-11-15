# Unchecked immutable values

`im.UncheckedValue[T]` holds a `*T`: Trust stores the pointer, and Unwrap returns it.
Use it when your package supplies immutable data that does not fit the checked forms:

- Your package promises not to change the data; the checker does not prove this.
- Only the package that declares `T` can wrap it with `im.Trust` or read it with `im.Unwrap`.
- Other packages can keep the handle and call your functions to read the data.

For example, package pattern can keep a compiled regexp behind a private
payload type:

```go
type compiled struct { re *regexp.Regexp }

func MustCompile(expression string) im.UncheckedValue[compiled] {
    return im.Trust(&compiled{re: regexp.MustCompile(expression)})
}

func Match(value im.UncheckedValue[compiled], text string) bool {
    return im.Unwrap(value).re.MatchString(text)
}
```

This package never changes or exposes the compiled regexp. Trust stores its
pointer; it does not freeze or copy the regexp.

## Allowed forms — exhaustive

A handle follows ordinary Go rules. It is Frozen and a read result, but not Copied.
Reading a handle through a projection does not expose its payload pointer. A marked object that stores it still
follows the [common forms](common-forms.md).

| Allowed form | Example |
| --- | --- |
| Use as an allowed stored type in a marked struct, a frozen collection, or ordinary Go | `type Pattern struct { im.Frozen; compiled im.UncheckedValue[compiled] }` |
| Copy, store, pass, or return the handle in any package | `other := p.compiled` |
| Use as a service result or Local field | `func (l *Loader) Current() im.UncheckedValue[Model]`; `type Reply struct { im.Local; Model im.UncheckedValue[Model] }` |
| Read a handle from a generated field or array getter | `handle := modelProjection.Handle()`; `handle := modelProjection.Handles().At(i)` |
| Read a handle or unnamed array of handles from a primitive projection | `handle := im.Value(projection)`; this also applies to Cell, slice, and map element projections. |
| Declare a zero handle; Unwrap returns nil | `var value im.UncheckedValue[compiled]` |
| In the package that declares the named type `T`, wrap a pointer | `value := im.Trust(&compiled{})` |
| In the package that declares the named type `T`, read the pointer | `value := im.Unwrap(p.compiled)` |
| In that package, use either function as a function value | `read := im.Unwrap[compiled]`; `value := read(p.compiled)` |

Aliases resolve to the original type and package. An instance of a named type,
such as `table[int]`, belongs to the package that declares `table`. Parentheses
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
| `p.compiled = im.Trust(&compiled{})` on a marked value | Construct a new marked value. |
