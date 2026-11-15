# Unchecked immutable values

`im.UncheckedValue[T]` holds a `*T`: Trust stores the pointer, and Unwrap returns it.
Use it when your package supplies immutable data that does not fit the checked forms:

- Your package promises not to change the data; the checker does not prove this.
- Only the package that declares `T` can wrap it with `im.Trust` or read it with `im.Unwrap`.
- Other packages can keep the handle and call your functions to read the data.

## Example: a compiled regular expression

A package can expose matching without exposing the mutable regexp object:

```go
package pattern

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
    "regexp"

    "github.com/lucasavila00/immutago/pkg/im"
)

type compiled struct { re *regexp.Regexp }

type Pattern struct {
    im.Frozen
    compiled im.UncheckedValue[compiled]
}

func MustCompile(expression string) *Pattern {
    re := regexp.MustCompile(expression)
    return &Pattern{compiled: im.Trust(&compiled{re: re})}
}

func Match(p *Pattern, text string) bool {
    return im.Unwrap(p.compiled).re.MatchString(text)
}
```

This package never changes or returns the compiled regexp. Trust stores its
pointer; it does not freeze or copy the regexp.

## Allowed forms — exhaustive

A persistent handle follows ordinary Go rules. A Projection cannot expose the handle
or call Unwrap. A marked object that stores it still
follows the [common forms](common-forms.md).

| Allowed form | Example |
| --- | --- |
| Use as an allowed stored type in a marked struct, a frozen collection, or ordinary Go | `type Pattern struct { im.Frozen; compiled im.UncheckedValue[compiled] }` |
| Copy, store, pass, or return the handle in any package | `other := p.compiled` |
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
