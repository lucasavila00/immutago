# Frozen strings

Use `im.FrozenString` for text that callers can store. Use
`im.FrozenByValueString` for a [local read view](local-views.md).
Both types hold immutable bytes. Neither type exposes a native string view,
byte slice view, substring view, or `String` method.

`FreezeString(text)` and `FreezeByValueString(text)` wrap the native string
without copying bytes. The input remains usable. A substring input can keep its
parent allocation alive. Use `Clone` when you need independent bytes.

Native strings and defined string types are rejected in immutable fields,
frozen type arguments, and [service results](service-results.md). Ordinary Go
inputs, local strings, and unmarked service state can still use `string`.

## Example: store text and return a view

```go
type Label struct {
    im.Shared
    Text im.FrozenString
}

func labelText(label Label) im.FrozenByValueString {
    return label.Text.ByValue()
}

func readLabel() {
    label := Label{Text: im.FreezeString("ready")}
    text := labelText(label)
    println(text.Len(), text.At(0))
    native := text.Clone()
    println(native)
}
```

The view shares the stored bytes. `native` has independent storage and follows
ordinary Go rules. A substring of `native` can keep that copy alive.

## String operations — exhaustive

Use these forms with the [common forms](common-forms.md) and the
[local view transport rules](local-views.md#transport--exhaustive).
Unlisted protected operations are rejected.

| Form | Shared string | Local string view |
| --- | --- | --- |
| Zero declaration or empty literal | `var text im.FrozenString`, `im.FrozenString{}` | `var text im.FrozenByValueString`, `im.FrozenByValueString{}` |
| Direct construction from a native string expression | `im.FreezeString(source[2:])` | `im.FreezeByValueString(source[2:])` |
| Construction in a marked literal, return, or direct typed call | `Label{Text: im.FreezeString(source)}` | `return im.FreezeByValueString(source)` |
| Copy, replace, or discard the whole value locally | `other := text`, `text = other`, `_ = text` | Same forms. |
| Read byte count or one byte | `text.Len()`, `text.At(0)` | Same forms. |
| Copy to a native string | `native := text.Clone()` | Same form. |
| Make a local view | `view := text.ByValue()` | Already a local view. |
| Compare with `==` or `!=`, or use a value switch | `same := text == other`, `switch text { case other: ... }` | Same forms with the same view type. |
| Pass to or return from a direct concrete function or method | `func read(text im.FrozenString) byte` | `func read(text im.FrozenByValueString) byte` |
| Alias or parentheses | `type Text = im.FrozenString`, `(text).Clone()` | `type Text = im.FrozenByValueString`, `(text).Clone()` |

Call methods on a local variable, field, or direct function result. `At` uses
byte indices and panics for an invalid index. Clone first for rune operations.
Equal byte sequences compare equal, including when their storage is separate.

Shared strings can be stored in marked fields, unnamed arrays, and shared frozen
collections. Local string views use the [local type and storage rules](local-views.md).
String constructors do not consume builders and do not need separate bindings.

## Rejected examples — non-exhaustive

| Form | Use instead |
| --- | --- |
| `type Label struct { im.Shared; Text string }` | Use `Text im.FrozenString`. |
| `func (c *Catalog) Text() string` | Return `im.FrozenByValueString` from `im.FreezeByValueString(c.text)`. |
| `im.FrozenMap[string, int]` | Use `im.FrozenMap[im.FrozenString, int]` and `im.FreezeString(key)`. |
| `text[1:]`, `text[0]`, or `len(text)` | Use `native := text.Clone(); part := native[1:]`, `text.At(0)`, or `text.Len()`. |
| `var value any = text` or `fmt.Print(text)` | Use `fmt.Print(text.Clone())`. |
| `read := text.Clone` or `im.FrozenString.Clone(text)` | Use `native := text.Clone()`. |
| `new(im.FrozenString)` or `&text` | Keep and pass the value. |
| `go read(text)`, `defer read(text)`, or a closure that captures text | Clone first; pass or capture the native string. |
| `type Other im.FrozenString; Other(text)` | Keep the library type or declare an alias. |
| `func read(text ...im.FrozenString)` | Pass an `im.FrozenSlice[im.FrozenString]` to a direct function. |

See [operation costs](performance.md) for Freeze, reads, and Clone costs.
