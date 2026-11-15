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
- Aliases keep the interface identity. Export the interface declaration when a
  Frozen record in another package uses it as a field. The generator uses the
  declared name; it does not select an exported alias for a private declaration.
- An interface can embed other interfaces when its complete method set meets
  these rules. Its method parameter and result types must have names that the
  generated file can use. Export private types inherited from another package.
- Declare only exported methods, apart from the private im seal. Method names
  Local and IsNil conflict with generated members and are rejected.
- Use fixed concrete parameters. Interface and function parameters are rejected.
  Ordinary inputs such as `[]float32` follow ordinary Go contracts. Protected
  inputs also need the [common call forms](common-forms.md#statements-and-calls).
- Return no results, or any number of results from the complete list below.

| Result type — exhaustive | Example |
| --- | --- |
| [Read result](types.md): Copied or Local | `Lookup(string) (float64, bool)`, `Name() im.StringProjection` |
| The predeclared `error` type or an alias | `Lookup(string) (float64, error)`, `Visit() error` |
| Native slice with Copied elements, including defined slices and aliases | `Lookup(key, buf []byte) (float64, bool, []byte)`, `Values() []Summary` where Summary embeds im.Copied |

No-result methods such as `Visit()` remain allowed. Result order is not restricted;
for example, `Read() (error, bool)` is allowed. Each result must satisfy one row.
A defined interface with an Error method is not the predeclared error type.
These method result types do not change the Frozen, Local, or service field rules.

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
| Bind several results to local names or blanks with `:=`, `=`, or `var` | `value, found, out := local.Lookup(key, buf)`, `value, found, out = local.Lookup(key, buf)`, `var value, found, out = local.Lookup(key, buf)` |
| Forward all results to a direct typed call or return | `use(local.Lookup(key, buf))`, `return local.Lookup(key, buf)` |

Use one binding per declaration or assignment, except for the result-call forms
above. Multiple-result bindings require local names or blanks for every destination.
Use the common forms for
statements, builders, fields, return values, and direct named calls. Each
implementation method must be declared on the exact converted type. Promoted
methods and implicit pointer dereferences do not qualify. The private seal
supplied by the embedded Frozen marker is the only exception.

Go interface equality and map-key rules still apply. A nil interface and an
interface containing a nil pointer are different values. IsNil checks the
interface itself. Calling a method on a nil interface has Go's usual panic.

Frozen interfaces can be service inputs and persistent Frozen fields. Exported
service methods return their generated local projection, not the persistent
interface. Methods retain the normal body checks and ordinary Go input contracts. For
example, an implementation can save its own Frozen receiver in a package
variable. A caller of its local projection cannot extract that receiver.
The checker does not infer method effects.

A slice result uses ordinary Go ownership. It can return a caller buffer such as
`buf[:n]`, a fresh slice, or an independent `Clone()` result. The checker does not
track which argument supplied it. The implementation's normal body checks still
reject native slices in Frozen fields and use of a builder after Freeze. A slice
result does not expose the implementation's frozen backing storage through an
allowed source operation. Ordinary inputs, errors, and dependency calls retain
Go contracts; this rule does not prove method isolation.

```go
type Tables interface {
    im.FrozenInterface
    Lookup(key, buf []byte) (float64, bool, []byte, error)
}

func (t *Tree) Lookup(key, buf []byte) (float64, bool, []byte, error) {
    buf = append(buf[:0], key...)
    return t.Weight, len(key) != 0, buf, nil
}
```

## Rejected examples — non-exhaustive

| Rejected | Allowed replacement |
| --- | --- |
| A foreign field uses `type Public = privateReader` | Export the original Reader declaration. |
| An inherited method returns a private type from another package | Export the result type in its declaring package. |
| An implementation with a mutable slice field | Store `im.FrozenSlice[T]` and freeze an exclusive builder. |
| `Scorer(nil)` | `var scorer Scorer = nil` |
| `var value any = scorer` | Keep the Scorer type. |
| `scorer.(*Tree)` or a type switch | Add an allowed method to Scorer. |
| `method := scorer.Score` or `Scorer.Score(scorer, features)` | `scorer.Score(features)` |
| `go scorer.Score(features)` or `defer scorer.Score(features)` | Call synchronously. |
| Capture scorer or its projection in a closure | Compute an ordinary result before the closure. |
| `Self() Scorer`, `Text() string`, or `Self() *Tree` | Return a local projection or a Copied result. |
| `Read() any`, `Read() func()`, or `Read() map[int]int` | Return listed concrete results; use a projection for a frozen map. |
| `Values() []string`, `Values() []*Tree`, or `Values() [][]byte` | Return a frozen collection projection, or a slice with Copied elements. |
| `value, outputs[0] = local.Read()` | Bind local names first, then assign the ordinary output separately. |
| Save ScorerProjection in a package variable or Frozen field | Save a Scorer value from an allowed construction. |
| Read the raw payload with `projection.Read()` | Use the generated method. |
| `chan Scorer` | Use direct calls with a Scorer parameter. |
