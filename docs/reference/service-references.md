# Service references

`supervise.Ref[T]` holds one stable service pointer. It uses the general [opaque handle](unchecked-values.md) shape:

```go
type Ref[T any] struct {
    im.Frozen
    value im.UncheckedValue[T]
}
```

Frozen and Local records can store the handle. The service itself has mutable state.

```go
worker := ref.Borrow()
first := worker.Count()
second := worker.Count()
```

## API

| Operation | Result |
| --- | --- |
| `ref.Borrow()` | The `*T` created by Start. Save it, return it, or call its methods using ordinary Go rules. |
| `ref.IsZero()` | Whether the reference is unset. Borrow panics on a zero Ref. |
| Copy a Ref | Another handle to the same service. |
| Stop the service | End its supervised work. The pointer stays valid; Stop does not prevent or wait for method calls. |

## Checker rules — exhaustive

| Operation | Rule |
| --- | --- |
| Create a service with Start | Go requires the service methods; the analyzer checks the [value embedding path](services.md#service-embedding--exhaustive). |
| Declare an exported service method | Its results follow the [service result signatures](service-results.md). |
| Store a Ref in protected im storage | Apply the [im rules](common-forms.md) to that storage. The handle does not make the service immutable. |
| Use Ref or the pointer returned by Borrow | Ordinary Go rules. Protected im fields keep their own rules. |

These examples use ordinary Go operations:

| Use | Example |
| --- | --- |
| Store or return the service pointer | `worker := ref.Borrow()`, `return worker` |
| Keep a method | `count := ref.Borrow().Count`, `borrow := ref.Borrow` |
| Use a method expression | `supervise.Ref[Worker].Borrow(ref)` |
| Use an embedded reference | `type Holder struct { supervise.Ref[Worker] }`; `holder.Borrow()` |
| Use interfaces or generic helpers | `var value any = ref`, `worker := identity(ref.Borrow())` |
| Call from a goroutine or closure | `go worker.Refresh()`, `func() int { return worker.Count() }` |

Service state changes, locks, and method inputs require normal Go review. Start
owns Init and Run. Do not copy an initialized service or Group; see the
[copy contract](services.md#copy-contract).

## References in im records

| Use | Frozen record | Local record |
| --- | --- | --- |
| Read a Ref field and save its Borrow result | Allowed. | Allowed. |
| Keep a bound Ref method | `borrow := record.Worker.Borrow` | `borrow := record.Worker.Borrow` |
| Capture the record in a closure | Allowed. | Copy the Ref or save its Borrow result before the closure. |
| Replace the Ref field | Rejected. | Rejected. |

Select embedded Ref fields explicitly inside Frozen or Local records. The ordinary
[immutable method rules](common-forms.md) still apply to those records.

## Rejected examples — non-exhaustive

| Rejected | Use instead |
| --- | --- |
| `record.Worker = other` on a Frozen or Local record | Construct another record. |
| `func() int { return local.Worker.Borrow().Count() }` captures a Local record | Save `worker := local.Worker.Borrow()` first; capture worker. |
| Store `*Worker` as a Frozen field | Store `supervise.Ref[Worker]`; the service pointer is mutable. |
