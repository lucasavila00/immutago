# Copied records

Embed `im.Copied` in a record whose data fields must copy by value:

```go
type Score struct {
    im.Copied
    Value float64
    Flags [2]bool
}

score := Score{Value: 2}
other := score
other.Value++
```

A value copy includes every data field. It shares no field storage with the
source. The checker proves the fields at the type declaration. Ordinary copies
can change independently. Copied records need no generated code.

## Fields — exhaustive

Embed exactly one `im.Copied` marker by value. Each data field must have one
of these types, including blank and embedded data fields:

| Allowed type | Example |
| --- | --- |
| Boolean or number, including a defined scalar type or alias | `OK bool`, `Value float64`, `Flags uint8` |
| Unnamed array of allowed fields, recursively | `Values [4]int`, `Grid [2][3]Score` |
| Another proved Copied record by value | `Score Score` |

Unlisted field types are rejected. Native strings,
FrozenString, pointers, slices, maps, channels, functions, interfaces, service
references, and other im handles are not Copied record fields. A Go string copy
shares its bytes. There is no owned-string field form.

[Opaque handles](unchecked-values.md), including service references, are read
results but are not Copied. Copies share their target.

## Operations — exhaustive

| Storage or use | Complete rule |
| --- | --- |
| Ordinary Copied value or ordinary holder | All Go operations are allowed. The examples below include mutation, pointers, methods, and transport. |
| Copied field or array element inside a Frozen or Local owner | Use the [common forms](common-forms.md). Read a value copy; do not write or take an address inside the owner. Unlisted protected operations are rejected. |
| Cell payload or frozen collection element | Use the fixed [Cell](cell.md), [frozen collection](frozen.md), and [projection](projections.md) APIs. Their reads can return independent Copied values. |
| Native slice or map passed to Freeze | Use an exclusive [builder](frozen.md#build-before-freezing). Freeze consumes it. |
| Exported service method result | Each result follows the [service result rules](service-results.md). There is no result count or order limit. |

An ordinary pointer can share its target under normal Go rules. The pointer is
not a Copied field type or a service result. The Copied proof applies to value
copies, not to pointer copies.

| Ordinary Go examples | Source |
| --- | --- |
| Declare, copy, or compare a value | `var score Score`, `other := score`, `score == other` |
| Change fields and array elements | `score.Value++`, `state.Result.Flags[0] = true`, `scores[0].Value = 2` |
| Use ordinary holders and native containers | `requestState{Result: score}`, `[]Score{score}`, `map[int]Score{1: score}`, `make(chan Score)` |
| Use pointers and pointer methods | `&score`, `new(Score)`, `func (s *Score) Set(v float64) { s.Value = v }`; `state.Result.Set(2)` |
| Bind or return several values | `x, y := pair(score)`; `func (s Score) Read() (float64, bool) { return s.Value, s.Flags[0] }` |
| Use interfaces, decoders, and type assertions | `var value any = &score`, `json.Unmarshal(data, &state)`, `value.(*Score).Set(2)` |
| Use generics, callbacks, and closures | `identity(score)`, `Box[Score]{Value: score}`, `callback(score)`, `func() { score.Value++ }` |
| Use method values or expressions | `set := score.Set`, `(*Score).Set(&score, 2)` |
| Use go, defer, or a function without a Go body | `go use(score)`, `defer state.release()`, `func decode(*Score)` |

A decoder needs no special permission. For example, a Copied record can implement
`json.Unmarshaler` with a pointer receiver. A decoder can call that method on an
ordinary holder's field. The implicit address in `payload.Ratio.UnmarshalJSON(data)`
is allowed.

Copy a protected field before changing it:

```go
score := frozen.Score
score.Value++
score.Set(2)

read := im.Value(cell.Load())
read.Value++
```

The original owner does not change. The same rule applies to generated getters,
`im.Value`, frozen collection value reads, and local collection Clone results.
Native Copied containers use ordinary alias rules unless Freeze requires an
exclusive builder. The checker adds no runtime ownership state.

## Rejected examples — non-exhaustive

| Rejected | Allowed replacement |
| --- | --- |
| `frozen.Score.Value++`, `local.Score.Flags[0] = true` | `score := frozen.Score; score.Value++`, or copy the Local field first. |
| `&frozen.Score`, `&local.Score.Flags[0]` | `score := frozen.Score; p := &score`, or copy the Local field first. |
| `frozen.Score.Set(2)` with a pointer receiver | `score := frozen.Score; score.Set(2)` |
| `frozen.Score = score` for a Frozen record | Construct a new complete Frozen record. |
| `struct { im.Copied; Text string }` | Use a Local result with StringProjection when text is needed. |
| `struct { im.Copied; Worker supervise.Ref[Worker] }` | Use a Local result with the reference and copied fields. |
| `struct { im.Copied; Score *Score }` | Use `Score Score`. |
| A named array field: `type Flags [2]bool` | Use `Flags [2]bool` directly in the record. |
| `im.CopyRecord(projection)` for `Projection[Score]` | `im.Value(projection)` |
| `alias := scores; frozen := im.FreezeSlice(scores)` | Construct a fresh exclusive builder, then Freeze it. |
| Use `scores` after `im.FreezeSlice(scores)` | Read the frozen result or Clone it into independent native storage. |

Copies have the normal Go struct and array copy cost. The marker adds no runtime
copy operation or allocation. See [operation costs](performance.md).
