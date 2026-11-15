# Scoped results

A **Scoped result** is an immutable value returned by value. The category includes
`int` and `string`; `im.Scoped` is the marker for a struct, not a requirement to
wrap each scalar in a struct.

Examples from the [configuration project](../examples/configuration/README.md):

| Method | Result |
| --- | --- |
| `FileCache.Text() string` | The last file contents. |
| `Limiter.Take() Decision` | A marked value with the request decision. |
| `Limiter.Usage() Usage` | A struct with `im.Scoped` and the current counts. |

```go
type Usage struct {
    im.Scoped
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
| Boolean, integer, floating-point, complex, or string value | `bool`, `int`, `float64`, `string` |
| Defined scalar | `type Count int` |
| Struct with `im.Scoped` or `im.Shared`, with proved immutable fields | `Usage`, a struct with `im.Shared` |
| Unnamed array of allowed stored elements | `[2]int`, `[2]Usage` |
| Frozen collection with allowed keys and elements | `im.FrozenSlice[Usage]`, `im.FrozenMap[string, int]` |
| Trusted immutable handle | `im.UncheckedValue[Model]` |
| Alias of an allowed type | `type Response = Usage` |

Every field and collection element follows the [stored-type rules](common-forms.md#markers-and-stored-types).
Those rules allow Shared pointers inside immutable values. A result itself cannot
be a pointer. Frozen results copy their handle, not their entries.

## Service methods

Every exported method on a service returns **exactly one Scoped result**.
This includes promoted user methods. Init, Run, and the runtime's own methods
are not message methods and are exempt.

This is the only service-specific analyzer rule. Start arguments, receivers,
closures, child groups, and ordinary method code use Go rules. Any protected
values used in that code still follow their im rules. In particular, Start saves
its argument: a Scoped value can be saved, but a Scoped pointer cannot.

## Rejected examples — non-exhaustive

| Result | Use instead |
| --- | --- |
| `*Usage`, or a pointer to a Shared struct | Return the value: `Usage` or `*p`. |
| `[]int` or `map[string]int` | Return a frozen collection. |
| `error`, `any`, or another interface | Return a marked result struct with the information the caller needs. |
| No result, or `(int, bool)` | Return one value; use a marked result struct for several fields. |
| An unmarked struct | Mark it and prove its fields. |

Scoped pointers still cannot be saved, returned, or captured. See
[Scoped objects](scoped.md) for their use inside a function.
