# Scoped results

A handler can return nothing or one **Scoped result**: a primitive, an array of
Scoped results, a struct with `im.Scoped`, or a service handle.

## Example: quota results

Examples from the [configuration project](../examples/configuration/README.md):

| Method | Result |
| --- | --- |
| `FileCache.Text() string` | The last file contents. |
| `Limiter.Take() Decision` | An integer type with the request decision. |
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
| Service handle | `supervise.Ref[Worker]`; see [handle operations](service-handles.md). |
| Empty Scoped marker value | `im.Scoped` |
| Struct with `im.Scoped` and proved immutable fields | `Usage` |
| Unnamed array of Scoped results | `[2]int`, `[2]Usage` |
| Alias of an allowed type | `type Response = Usage` |

Fields of a Scoped struct follow the [stored-type rules](common-forms.md#markers-and-stored-types).
They can contain frozen collections, Shared values, Shared pointers, and service handles.
Callers can read or copy the result; the checker rejects field writes and storing
or capturing a Scoped pointer.

## Service methods

Every exported user method on a service uses one of these result forms:

| Allowed forms — exhaustive | Example |
| --- | --- |
| No result | `func (c *Counter) Pause() { c.paused.Store(true) }` |
| One Scoped result | `func (l *Limiter) Usage() Usage` |

This includes promoted user methods. Init, Run, and runtime methods such as
`Args` and `Children` are exempt.

Start arguments, receivers, closures, child groups, and ordinary method code use
Go rules. Protected values follow the [im rules](common-forms.md). Start saves its
argument: a Scoped value can be saved, but a Scoped pointer cannot.

## Rejected examples — non-exhaustive

| Result | Use instead |
| --- | --- |
| `*Usage` | Declare a value result such as `Read() Usage`; return `*p`. |
| `[]int`, `map[string]int`, or a frozen collection | Put a frozen collection in a struct with `im.Scoped` and return that struct. |
| `error`, `any`, or another interface | Return a struct with `im.Scoped` and the information the caller needs. |
| `(int, bool)` | Return a struct with `im.Scoped` and fields for both values. |
| An unmarked struct, Shared struct or pointer, or `im.UncheckedValue[T]` | Return a struct with `im.Scoped` and allowed fields. |
| `type Counts [2]int` | Return `[2]int`, or a struct with `im.Scoped` and an array field. |

Scoped pointers cannot be saved, returned, or captured. See
[Scoped objects](scoped.md) for their use inside a function.
