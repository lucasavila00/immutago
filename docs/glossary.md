# Glossary

## Immutable data

| Term | Meaning |
| --- | --- |
| **Immutable object** | An object whose fields cannot change after construction. See [common forms](common-forms.md). |
| **Marker; marked struct** | `im.Shared` or `im.ByValue`, embedded by value; a struct with one of these markers and allowed field types. See [stored types](common-forms.md#markers-and-stored-types). |
| **Shared object** | A read-only struct with `im.Shared`. Callers can keep its values and pointers. See [Shared](shared.md). |
| **ByValue object** | A read-only struct with `im.ByValue`. Callers keep value copies and use pointers only through the [local pointer forms](by-value.md). |
| **Builder** | A local Go slice or map whose storage is checked for exclusive use. See [build before freezing](frozen.md#build-before-freezing). |
| **Freeze** | `im.FreezeSlice` or `im.FreezeMap` transfers a builder's storage to a frozen collection without copying entries. See [frozen collections](frozen.md). |
| **Frozen collection; Clone** | A read-only `im.FrozenSlice` or `im.FrozenMap`; its `Clone` method makes an independent mutable slice or map with the same entries. See [collection reads](frozen.md#reads-and-clone). |
| **Owner slot; Owner holder** | An `im.Owner[T]` package variable or named field; an unmarked struct that contains such a field. See [Owner](owner.md). |
| **Unchecked immutable value** | An `im.UncheckedValue[T]` handle. The package that declares `T` promises that the wrapped data never changes. See [unchecked values](unchecked-values.md). |

## Services

| Term | Meaning |
| --- | --- |
| **Service** | A struct that embeds `supervise.Service[A]`, directly or through embedded fields. The runtime calls Init for setup and Run for background work. See [services](services.md). |
| **Service method** | A method declared on a service or promoted from an embedded field. Exported methods have a [result rule](service-results.md), except Init, Run, and methods supplied by the runtime. |
| **Service result** | The optional single result of an exported service method. Its [allowed types](service-results.md#result-types--exhaustive) include scalars and ByValue structs. |
| **Service reference** | A `supervise.Ref[T]` that can be copied, stored, and returned. `ref.Current().Method(args)` calls the current service. See [service references](service-references.md). |
| **Service run** | One service value's background execution and child group. A restart creates a new value and run. References to old children do not select children from the new run. See [child lifetimes](service-children.md#child-lifetimes). |
| **Group; child** | A `supervise.Group` manages services and nested groups. Each service or nested group is a child of its parent group. See [child services](service-children.md). |

## Checker terms

| Term | Meaning |
| --- | --- |
| **Protected storage** | Marked objects and their fields, frozen collections, and arrays containing them. The [source rules](common-forms.md) check operations on this storage and its pointers. |
| **Consumed builder** | A builder after Freeze. Further uses of its original slice or map are rejected. See [frozen collections](frozen.md). |
| **Checked package** | A package selected for im checks, service result checks, or dependency failure reports. See the [package boundary](rules.md#package-boundary). |
| **Allowed form; matcher** | A documented Go expression or statement shape; the analyzer code that checks its syntax, resolved types, and context. See the [analysis pipeline](arch/pipeline.md#source-forms-and-dependency-failures). |
