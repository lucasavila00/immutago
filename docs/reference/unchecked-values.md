# UncheckedValue

`im.UncheckedValue[T]` stores a `*T`. The package that declares T controls access
and synchronization. The target can be mutable. UncheckedValue cannot contain
Local data, including through nested fields or arrays.

## Forms — exhaustive

| Form | Rule |
| --- | --- |
| `im.UncheckedValue[T]` | Frozen and a read result. Only the package that declares T can use Trust or Unwrap. |
| Named Frozen record with one named UncheckedValue field and no other data fields | Frozen and a read result. Generic declarations and aliases work. The declaration can be at package scope, inside a function, or in a test file. No projection generation is needed. |
| Unnamed pointer to that Frozen record, including aliases | Frozen and a read result. Use the [Frozen pointer forms](frozen-records.md#stored-pointers--exhaustive). |

```go
type counter struct { value atomic.Int64 }

type Counter struct {
    im.Frozen
    state im.UncheckedValue[counter]
}

func NewCounter() Counter { return Counter{state: im.Trust(&counter{})} }
func (c Counter) Add(n int64) int64 { return im.Unwrap(c.state).value.Add(n) }
```

Counter's state field cannot be replaced. Add changes its target. An extra data
field makes Counter subject to the normal Frozen generation and result rules.
Neither Counter nor UncheckedValue is Copied.

## Operations — exhaustive

| Operation | Rule |
| --- | --- |
| Copy, store, pass, return, or capture UncheckedValue or the Frozen record above | Ordinary Go value storage and transport. Copies share the target. Fields inside a protected owner keep that owner's rules. |
| Use these types in a Frozen field, Local field, service result, or frozen collection | Use the [type rules](types.md), including unnamed arrays. |
| Read through a generated getter or `im.Value(projection)` | Return the stored value. No target copy. |
| Declare a zero UncheckedValue | Allowed. Unwrap returns nil. |
| `im.Trust(pointer)` | T must be a named type declared in this package. Store its pointer. |
| `im.Unwrap(value)` | Same package rule. Return the stored pointer. The package may expose it through its API. |
| Select Trust or Unwrap as a function value | Same type and package rules. |
| Replace a variable or field in an ordinary holder | Replace the whole value. |
| Call methods, including promoted methods | Value receivers copy the record. Pointer receivers need an existing Frozen pointer; a call cannot take the address of an existing Frozen record. |
| Take the address of UncheckedValue in ordinary storage | `&raw`, `&holder.Raw`, or `&values[i]`. |
| Allocate a pointer to the Frozen record | Fresh storage: `&Counter{}`, `new(Counter)`, or `new(value)`. |

Aliases resolve to the original type and package. A named instance such as
`table[int]` belongs to the package that declares table. UncheckedValue has no
reader methods. Target mutation, synchronization, and pointer exposure follow
the declaring package's Go API contract.

## Rejected examples — non-exhaustive

| Rejected | Use instead |
| --- | --- |
| `im.Trust(&engine.Table{})` outside package engine | Call an engine constructor. |
| `im.Unwrap(table)` outside the package that declares Table | Call a method or function supplied by that package. |
| `func Wrap[T any](v *T) im.UncheckedValue[T] { return im.Trust(v) }` | Use a named payload type declared by this package. |
| `im.Trust(&rows)` for a native slice variable | Declare a named payload type. |
| `c.state = im.Trust(&counter{})` | Construct another Counter. |
| `p := &value`, `p := &holder.Counter`, or `p := &counters[i]` for Counter | Use `new(value)`, `new(holder.Counter)`, or `new(counters[i])`. Each copies the record and keeps the same target. |
| Store UncheckedValue or Counter inside a Copied record | Use a Local result or an ordinary holder. |
| `holder.Pointer()` when Holder embeds Counter by value and Pointer has a `*Counter` receiver | Embed `*Counter` and construct it with `new(NewCounter())`, or use a value receiver. |
