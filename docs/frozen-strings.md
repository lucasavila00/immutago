# Frozen strings

Use `im.FrozenString` for text in immutable fields, frozen collection keys or
elements, and exported service method results. Native strings, including defined
string types, are rejected in these positions. Ordinary Go inputs, local strings,
and unmarked service state can still use `string`.

`im.FreezeString(text)` copies the bytes into independent storage. If `text` is
a substring, the frozen value does not retain its larger parent allocation.
The input remains usable. `Clone()` copies the bytes into a native string.
The API has no substring view, native string view, byte slice view, or `String`
method.

```go
package main

import "github.com/lucasavila00/immutago/pkg/im"

type Label struct {
    im.Shared
    Text im.FrozenString
}

func main() {
    source := "prefix: label"
    label := Label{Text: im.FreezeString(source[8:])}
    println(label.Text.Len(), label.Text.At(0))
    text := label.Text.Clone()
    println(text, source)
}
```

## Allowed forms — exhaustive

Use these forms with the [common statement and transport forms](common-forms.md).
The list is complete. Unlisted protected operations are rejected.

| Form | Example |
| --- | --- |
| Zero declaration or empty literal | `var text im.FrozenString`, `im.FrozenString{}` |
| Direct construction from a native string expression | `text := im.FreezeString(source[2:])`, `return im.FreezeString(source)` |
| Construction in a marked literal or direct typed call | `Label{Text: im.FreezeString(source)}`, `read(im.FreezeString(source))` |
| Copy, replace, or discard the whole value | `other := text`, `text = other`, `_ = text` |
| Read bytes or byte count through a direct method call | `text.At(0)`, `text.Len()` |
| Copy to a native string through a direct method call | `native := text.Clone()` |
| Compare values with `==` or `!=` | `same := text == other` |
| Use a value switch | `switch text { case other: ... }` |
| Store in a marked field, unnamed array, or frozen collection | `Label{Text: text}`, `[2]im.FrozenString{text}`, `im.FrozenSlice[im.FrozenString]` |
| Use as a frozen map key | `im.FrozenMap[im.FrozenString, int]` |
| Pass to or return from a direct concrete function or method | `func read(text im.FrozenString) string { return text.Clone() }` |
| Return from an exported service method | `func (c *Catalog) Text() im.FrozenString { return im.FreezeString(c.text) }` |
| Alias and parentheses | `type Text = im.FrozenString`, `(text).Clone()` |

`At` uses byte indices and panics for an invalid index. For rune operations,
clone first, then use ordinary Go string operations. Equal byte sequences compare
equal, even if they have separate storage. A frozen copy shares the whole frozen
allocation; it cannot select a smaller view of it.

Native slices and maps with frozen strings use the [builder rules](frozen.md).
`FreezeString` does not consume a builder and does not need a separate binding.

## Rejected examples — non-exhaustive

| Form | Use instead |
| --- | --- |
| `type Label struct { im.Shared; Text string }` | `type Label struct { im.Shared; Text im.FrozenString }` |
| `func (c *Catalog) Text() string` | Return `im.FrozenString`, constructed with `im.FreezeString(c.text)`. |
| `im.FrozenMap[string, int]` | `im.FrozenMap[im.FrozenString, int]`; construct each key with `im.FreezeString(key)`. |
| `text[1:]`, `text[0]`, or `len(text)` | `native := text.Clone(); part := native[1:]`, `text.At(0)`, or `text.Len()`. |
| `var value any = text` or `fmt.Print(text)` | `fmt.Print(text.Clone())` |
| `read := text.Clone` or `im.FrozenString.Clone(text)` | `native := text.Clone()` |
| `new(im.FrozenString)` or `&text` | Keep and pass the frozen value. |
| `go read(text)`, `defer read(text)`, or a closure that captures text | Clone first and pass or capture the native string. |
| `type Other im.FrozenString; Other(text)` | Keep the `im.FrozenString` type or declare an alias. |
| `func read(text ...im.FrozenString)` | Pass an `im.FrozenSlice[im.FrozenString]` to a direct function. |

## Costs and limits

`FreezeString` and `Clone` take O(n) time and space. A nonempty copy that escapes
uses one allocation; empty copies need none. `Len` and `At` take constant time
with no allocation. Go can omit storage for results that do not escape.

A substring of a cloned native string can retain that clone's allocation. It
cannot retain the frozen allocation through that copy. The guarantee covers
this API's byte storage. It does not prove service isolation or inspect the
payload of `im.UncheckedValue`. See [operation costs](performance.md).
