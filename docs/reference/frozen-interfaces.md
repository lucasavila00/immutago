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
projection returns a ScorerProjection from its Scorer getter. The generated projection forwards declared interface methods. It has no raw
payload getter. A declared method can return a Frozen value, pointer, or interface
under the ordinary method rules.

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
- Use fixed or variadic parameters. Ordinary inputs, including interface and function values,
  follow ordinary Go contracts. Protected inputs also need the
  [common call forms](common-forms.md#statements-and-calls).
- Method results use the same type, return, and transport rules as ordinary named
  functions and concrete methods. There is no interface-specific result category.

| Parameter form — exhaustive | Example |
| --- | --- |
| No parameters | `Visit()` |
| Fixed parameters with ordinary Go types, including interfaces and functions | `Score([]float32) float64`, `Apply(value any, callback func() int) int`, `Read(interface{ Number() int }) int` |
| Variadic ordinary parameters, with zero, individual, or spread arguments | `Sum(...int) int`; `reader.Sum()`, `reader.Sum(1, 2)`, `reader.Sum(values...)` |
| Variadic parameters after fixed parameters | `Read(prefix string, values ...int) int`; `reader.Read("key", 1, 2)` |
| Fixed protected parameters under the common call forms | `Read(im.StringProjection) int`, `Read(*LocalRecord) int`, `WithScorer(Scorer) int` |

Named types, aliases, and named or unnamed parameters use the same rules.
A variadic parameter follows the native-slice rules for its element type. Native
slices of protected values remain rejected. Generated methods keep the variadic
signature and pass the final slice with `...`.
Implementation methods can retain ordinary inputs. A callback contract
cannot carry protected storage. An argument cannot box protected data or capture
it in a closure. These rules also apply to generated projection calls and concrete
implementation methods. The checker does not infer callback effects.

| Result form — exhaustive | Example |
| --- | --- |
| No results | `Visit()` |
| One result | `Score([]float32) float64`, `Name() string`, `Self() *Tree` |
| Several results, with any result order | `Lookup(key, buf []byte) (float64, bool, []byte, error)`, `Self() (*Tree, Scorer)` |
| Named results in either result form | `Lookup([]byte) (value float64, found bool)` |

Each result keeps its normal Go and im rules, including aliases and defined
types. A method can return Frozen data or Local values. Local pointers cannot be
returned, and method bodies cannot box protected data in ordinary interfaces or
capture protected object values in closures. Frozen pointer handles can be boxed,
captured, or stored in ordinary native containers. See the complete
[common forms](common-forms.md#statements-and-calls).
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
variable. A declared method can return that receiver as a Frozen pointer or interface.
The returned value keeps its Frozen storage rules.
The checker does not infer method effects.

A slice result can return a caller buffer such as `buf[:n]`, a fresh slice, or an
independent `Clone()` result. No argument-origin analysis is required. Normal
body checks still reject native slices in Frozen fields and use of a builder
after Freeze. Ordinary inputs, results, and dependency calls use Go contracts.
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
| `Apply(...im.StringProjection)` | Use a direct `Apply(im.StringProjection)` parameter. |
| `reader.Apply(local, nil)` with `local im.StringProjection` | Pass `local.Unsafe()`, or declare a direct `im.StringProjection` parameter. |
| `reader.Apply(nil, func() int { return local.Len() })` | Read `n := local.Len()` before the closure, then return `n`. |
| `Apply(func() im.StringProjection)` | Call a named function directly, then pass its projection to a direct typed method. |
| `scorer.(*Tree)` or a type switch | Add an allowed method to Scorer. |
| `method := scorer.Score` or `Scorer.Score(scorer, features)` | `scorer.Score(features)` |
| `go scorer.Score(features)` or `defer scorer.Score(features)` | Call synchronously. |
| Capture scorer or its projection in a closure | Compute an ordinary result before the closure. |
| `Self() *LocalRecord` | Return the Local record value. |
| `Values() []Tree` | Return `[]*Tree`, a FrozenSlice, or a projection. |
| `func (t Tree) Value() any { return t }` | Declare a `*Tree` or Scorer result. |
| `func (t Tree) Callback() func() float64 { return func() float64 { return t.Weight } }` | Copy Weight to an ordinary local before the closure. |
| `value, outputs[0] = local.Read()` | Bind local names first, then assign the ordinary output separately. |
| Save ScorerProjection in a package variable or Frozen field | Save a Scorer value from an allowed construction. |
| `chan Scorer` | Use direct calls with a Scorer parameter. |

A generic `im.Projection[Scorer]` can return its Frozen interface with
`projection.Unsafe()`. The result keeps the Frozen interface rules above. It
does not permit interface assertions or other unlisted operations.
