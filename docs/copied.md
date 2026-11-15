# Copied records

Embed `im.Copied` in a record that a service can return by value:

```go
type Score struct {
    im.Copied
    Value float64
    Flags [2]bool
}

type Scorer struct {
    supervise.Service[supervise.NoArgs]
    last Score
}

func (s *Scorer) Last() Score { return s.last }

type requestState struct {
    Result Score
    Attempts int
}

state := requestState{Result: scorer.Last()}
state.Attempts++
state.Result = Score{Value: 3} // Replace the complete stored value.
```

A copy includes every data field. It shares no field storage with the source.
The checker proves the fields at the type declaration. Fields cannot change
after construction, including fields in a local copy. Use a new complete value
to change a result. Copied records need no generated code.

## Fields — exhaustive

Embed exactly one `im.Copied` marker by value. Each data field must have one
of these types, including blank and embedded data fields:

| Allowed type | Example |
| --- | --- |
| Boolean or number, including a defined scalar type or alias | `OK bool`, `Value float64`, `Flags uint8` |
| Unnamed array of allowed fields, recursively | `Values [4]int`, `Grid [2][3]Score` |
| Another proved Copied record by value | `Score Score` |

This list is complete. Unlisted field types are rejected. In particular, native
strings, FrozenString, pointers, slices, maps, channels, functions, interfaces,
service references, and other im handles are not Copied record fields. A Go
string copy shares its bytes. There is no owned-string field form.

The general [Copied category](types.md) also permits opaque service references
and arrays of those references. That existing handle rule does not permit
reference fields in a Copied record.

## Operations — exhaustive

Use these forms with the [common statement and call forms](common-forms.md).
Unlisted operations on Copied storage are rejected.

| Allowed form | Example |
| --- | --- |
| Named, local, or anonymous record declaration; defined record or alias | `type Other Score`, `type Alias = Score`, `struct { im.Copied; N int }` |
| Zero value, literal, copy, whole variable replacement, or discard | `var score Score`, `score := Score{Value: 1}`, `other := score`, `score = Score{Value: 2}`, `_ = score` |
| Field or array-element read; comparable value equality | `value := score.Value`, `flag := score.Flags[0]`, `same := score == other` |
| Store by value in an ordinary struct, array, native slice, or native map | `requestState{Result: score}`, `[2]Score{score}`, `[]Score{score}`, `map[int]Score{1: score}` |
| Replace a complete field or element in an ordinary mutable holder | `state.Result = score`, `scores[0] = score`, `byID[1] = score` |
| Change an ordinary field beside a Copied field | `state.Attempts++` |
| Construct or use a pointer to an ordinary holder | `state := &requestState{Result: score}`, `state.Result = other` |
| Create, copy, slice, read, or change a native container handle | `scores := make([]Score, 2)`, `alias := scores`, `scores = append(scores, score)`, `part := scores[1:]` |
| Use native container builtins | `len(scores)`, `cap(scores)`, `copy(scores, otherScores)`, `delete(byID, 1)`, `clear(byID)` |
| Pass or return through direct concrete functions and methods | `read(score)` with `func read(score Score) int`; `func (s *Scorer) Last() Score` |
| Declare and call an exact value-receiver method | `func (s Score) Number() float64 { return s.Value }`; `score.Number()` |
| Use as a Local or Frozen field, Cell payload, or frozen collection argument | `struct { im.Local; Score Score }`, `im.Cell[Score]`, `im.FrozenSlice[Score]` |
| Read through the fixed copy and projection APIs | `score := im.Copy(projection)`, `score := settingsProjection.Score()` |
| Clone a local slice or map projection with Copied arguments | `scores := projection.Clone()` |
| Freeze an exclusive native builder in a separate binding | `frozen := im.FreezeSlice(scores)` |

An ordinary holder can change. Its stored Copied record's fields cannot change.
A Copied field inside a Frozen or Local record also keeps that owner's write
restriction: replacing the field is rejected. Copy the field into an ordinary
binding to use it separately.

Native Copied containers use ordinary alias rules until Freeze requires an
exclusive builder. Freeze still consumes that builder. Copied values and their
holders keep the concrete-call, interface, generic, closure, go, and defer rules
in the common forms. Pointers to Copied records and channels of Copied values
are not supported.

## Rejected examples — non-exhaustive

| Rejected | Allowed replacement |
| --- | --- |
| `score.Value++`, `score.Flags[0] = true` | `score = Score{Value: score.Value + 1, Flags: score.Flags}` |
| `state.Result.Value = 2` | `state.Result = Score{Value: 2}` |
| `scores[0].Value = 2` | `scores[0] = Score{Value: 2}` |
| `frozen.Score = score` for a Frozen record | Construct a new complete Frozen record. |
| `&score`, `&state.Result`, `new(Score)`, `func (s *Score) Change()` | Pass, return, or read a Score value. |
| `struct { im.Copied; Text string }` | Use a Local result with StringProjection when text is needed. |
| `struct { im.Copied; Worker supervise.Ref[Worker] }` | Use a Local result with the reference and copied fields. |
| A named array field: `type Flags [2]bool` | Use `Flags [2]bool` directly in the record. |
| `var boxed any = score`, `identity(score)` through a user generic | Keep the concrete Score type and call a direct typed function. |
| `callback := func() float64 { return score.Value }` | Copy `value := score.Value` first, then capture value. |
| `im.CopyRecord(projection)` for `Projection[Score]` | `im.Copy(projection)` |

Copies have the normal Go struct and array copy cost. The marker adds no runtime
copy operation, allocation, or ownership state. See [operation costs](performance.md).
