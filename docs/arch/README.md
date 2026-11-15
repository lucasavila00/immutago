# Architecture

The analyzer checks types, writes, and builder use. The runtime shares native Go
storage; it adds no ownership flags.

## From an update to a read

```go
type Rates struct {
    im.Frozen
    Values im.FrozenSlice[int]
}

var current RatesCell // Generated with RatesProjection.
```

```text
native slice ──FreezeSlice──> FrozenSlice ──Store Rates──> Cell
                                                          │ Load
                                                          ▼
       int <──Value── element projection <──At── local projection
```

| Stage | Go operation | What happens |
| --- | --- | --- |
| Build | `values := []int{10, 20}` | Create one local builder. |
| Freeze | `frozen := im.FreezeSlice(values)` | Prove the element type; consume `values`; share its backing array. |
| Publish | `current.Store(Rates{Values: frozen})` | Copy the root and replace an atomic pointer. |
| Select | `view := current.Load()` | Keep the selected root. A later Store does not change this view. |
| Read | `price := im.Value(view.Values().At(0))` | Read through the root and copy one integer. |

## Implementation

| Follow this work | Page |
| --- | --- |
| Select a package, prove types, match operations, report failures | [Analysis pipeline](pipeline.md) |
| Consume a builder, publish a root, generate field readers | [Storage and ownership](ownership.md) |
| Publish a service, retry it, stop its children | [Services](services.md) |
| Check runtime assumptions and analyzer decisions | [Validation](validation.md) |
| Measure runtime operations and analyzer data structures | [Performance](performance.md) |

The [language reference](../reference/rules.md) defines the allowed Go forms.
Service locks, state changes, and input use still need review in service code.
