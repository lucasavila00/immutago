# Opaque handles

An opaque handle keeps a pointer. Its package controls access to the target.
The target can be mutable; review its API and synchronization in that package.

## Forms — exhaustive

| Form | Meaning |
| --- | --- |
| `im.UncheckedValue[T]` | The raw opaque handle. Only the package that declares T can call Trust or Unwrap. |
| A named struct with `im.Frozen` and one named `im.UncheckedValue[T]` field | A handle with its own API. Generic types and aliases work. Named declarations can be at package scope, local, or in test files. No projection generation is needed. |

```go
type counter struct { value atomic.Int64 }

type Counter struct {
    im.Frozen
    state im.UncheckedValue[counter]
}

func NewCounter() Counter {
    return Counter{state: im.Trust(&counter{})}
}

func (c Counter) Add(n int64) int64 {
    return im.Unwrap(c.state).value.Add(n)
}
```

Counter's state field cannot be replaced. Add changes the target through an
atomic operation. Trust adds no lock, copy, or allocation; NewCounter allocates
the counter itself.

## Operations — exhaustive

| Operation | Rule |
| --- | --- |
| Copy, store, pass, return, or capture a handle | Ordinary Go value storage and transport. Copies share the target. |
| Use a handle as a Frozen field, Local field, service result, or frozen collection element | Allowed, including unnamed arrays of handles. |
| Read a handle through a generated getter or `im.Value(projection)` | Return the handle itself. No target copy. |
| Declare a zero handle | Allowed. Unwrap returns nil for its zero payload. |
| `im.Trust(pointer)` | T must be a named type declared in this package. Store its pointer. |
| `im.Unwrap(value)` | Same package rule. Return the stored pointer. The package may expose it through its API. |
| Change a handle variable or an ordinary holder's handle field | Replace the whole handle. |
| Call methods, including promoted methods | Value receivers copy the handle. Pointer receivers need an existing Frozen pointer; calls must not take the address of a replaceable handle field. |
| Use a pointer to the Frozen handle | Apply the normal [Frozen pointer rules](frozen-records.md#stored-pointers--exhaustive). |

Handles and pointers to their Frozen wrappers are Frozen and read results. They are not Copied: copying one keeps an
edge to its target. Their payload cannot contain Local data. An extra field makes
a Frozen struct an ordinary Frozen record, with its normal generation and result rules.

Aliases resolve to the original type and package. A named instance such as
`table[int]` belongs to the package that declares table. Trust and Unwrap can be
used as function values in that package. UncheckedValue has no reader methods.

| Responsibility | Checked by |
| --- | --- |
| Handle shape, fixed fields, and absence of Local payloads | Analyzer |
| Package allowed to call Trust or Unwrap | Analyzer |
| Target mutation, synchronization, and pointer exposure | Declaring package's API and code review |

## Rejected examples — non-exhaustive

| Rejected | Use instead |
| --- | --- |
| `im.Trust(&engine.Table{})` outside package engine | Call an engine constructor. |
| `im.Unwrap(table)` outside the package that declares Table | Call a method or function supplied by that package. |
| `func Wrap[T any](v *T) im.UncheckedValue[T] { return im.Trust(v) }` | Wrap a named payload type declared by this package. |
| `im.Trust(&rows)` for a native slice variable | Declare a named payload type. |
| `c.state = im.Trust(&counter{})` | Construct another Counter. |
| Store a handle inside an im.Copied record | Use a Local result or an ordinary holder. |
| `holder.Pointer()` when Holder embeds Counter by value and Pointer has a `*Counter` receiver | Embed `*Counter` and construct it with `new(counter)`, or use a value receiver. |
