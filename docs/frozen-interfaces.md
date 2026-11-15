# Frozen interfaces

Embed `im.FrozenInterface` in a named interface to store polymorphic Frozen data.
The compiler seal is necessary, but it does not prove the implementation's
fields. Immutago checks the concrete type at each conversion.

```go
type Scorer interface {
    im.FrozenInterface
    Score(features []float32) float64
}

type Tree struct {
    im.Frozen
    Weight float64
}

func (t *Tree) Score(features []float32) float64 {
    return t.Weight * float64(len(features))
}

type Model struct {
    im.Frozen
    Scorer Scorer
}
```

The package uses the [required generation step](rules.md#required-generation).
It generates `ScorerProjection` as well as the record projections. A Model
projection returns a ScorerProjection from its Scorer getter. The interface
payload cannot leave that local projection.

## Interface declarations — exhaustive

- Declare a named, non-generic interface at package scope in a non-test Go file.
- Embed `im.FrozenInterface`, directly or through another Frozen interface.
- Aliases keep the interface identity. An interface can embed other interfaces
  when its complete method set meets these rules.
- Declare only exported methods, apart from the private im seal.
- Use fixed concrete parameters. Interface and function parameters are rejected.
  Ordinary inputs such as `[]float32` follow ordinary Go contracts. Protected
  inputs also need the [common call forms](common-forms.md#statements-and-calls).
- Return no result or one [read result](types.md). For example, `Score([]float32)
  float64`, `Name() im.StringProjection`, or `Visit()`.

These lists are complete. Unlisted operations on protected storage are rejected.

## Values and calls — exhaustive

| Allowed form | Example |
| --- | --- |
| Declare zero or nil | `var scorer Scorer`, `var scorer Scorer = nil` |
| Convert a proved Frozen record value or pointer | `scorer := Scorer(&Tree{Weight: 2})` |
| Assign, pass, or return a proved implementation | `var scorer Scorer = &Tree{Weight: 2}`; `use(scorer)`; `return scorer` |
| Copy or convert between proved Frozen interfaces | `other := scorer`, `other = scorer`, `base := Base(scorer)` |
| Replace the whole interface binding | `scorer = &Tree{Weight: 3}`, `scorer = nil` |
| Store in Frozen fields, arrays, Cells, or frozen collections | `Model{Scorer: scorer}`, `im.FrozenSlice[Scorer]` |
| Call a method directly | `score := scorer.Score(features)` |
| Compare with `==` or `!=` | `scorer == nil`, `scorer == other` |
| Wrap a local projection | `local := NewScorerProjection(im.ProjectionOf(scorer))` |
| Call through the local projection | `score := local.Score(features)`, `empty := local.IsNil()` |

Use one binding per declaration or assignment. Use the common forms for
statements, builders, fields, return values, and direct named calls. Each
implementation method must be declared on the exact converted type. Promoted
methods and implicit pointer dereferences do not qualify. The private seal
supplied by the embedded Frozen marker is the only exception.

Go interface equality and map-key rules still apply. A nil interface and an
interface containing a nil pointer are different values. IsNil checks the
interface itself. Calling a method on a nil interface has Go's usual panic.

Frozen interfaces can be service inputs and persistent Frozen fields. Exported
service methods return their generated local projection, not the persistent
interface. Methods retain the normal body checks and ordinary Go input
contracts; this feature does not prove purity or whole-program isolation.

## Rejected examples — non-exhaustive

| Rejected | Allowed replacement |
| --- | --- |
| An implementation with a mutable slice field | Store `im.FrozenSlice[T]` and freeze an exclusive builder. |
| `Scorer(nil)` | `var scorer Scorer = nil` |
| `var value any = scorer` | Keep the Scorer type. |
| `scorer.(*Tree)` or a type switch | Add an allowed method to Scorer. |
| `method := scorer.Score` or `Scorer.Score(scorer, features)` | `scorer.Score(features)` |
| `go scorer.Score(features)` or `defer scorer.Score(features)` | Call synchronously. |
| Capture scorer or its projection in a closure | Compute an ordinary result before the closure. |
| `Self() Scorer`, `Text() string`, or `Values() []int` | Return a local projection or a copied result. |
| Save ScorerProjection in a package variable or Frozen field | Save a Scorer value from an allowed construction. |
| Read the raw payload with `projection.Read()` | Use the generated method. |
| `chan Scorer` | Use direct calls with a Scorer parameter. |
