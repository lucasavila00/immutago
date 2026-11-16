# Glossary

| Term | Meaning |
| --- | --- |
| Frozen | Immutable stored data with the recursive [Frozen field rule](types.md). Copies can share backing storage. |
| Frozen record | A struct with `im.Frozen`. |
| Copied | Boolean/numeric data, records with `im.Copied`, and unnamed arrays of Copied values. Frozen records are not Copied. |
| Copied record | A struct with `im.Copied`. All data fields copy by value. Ordinary copies can change independently. It has no string or reference fields. |
| Local value | A read projection, a record with `im.Local`, or an unnamed array of Local values. It cannot enter persistent storage. |
| Read result | Copied, Local, opaque handles, or unnamed arrays of read results. This is a type category, not a service-specific field rule. |
| Projection | Local access to a Frozen value. Generated types such as SettingsProjection return Copied fields, unchecked handles, or more projections. |
| Cell; cell holder | An `im.Cell[T]` slot; an unmarked struct containing a named slot or holder field. Generated types such as SettingsCell are holders. Holder pointers can be shared; holder values cannot be copied. |
| Builder | A native slice or map checked for exclusive use before Freeze. |
| FrozenString; FrozenSlice; FrozenMap | Stored string, slice, and map types. Copies share immutable backing storage. |
| Freeze | Wrap native storage without a copy. FreezeSlice and FreezeMap consume a builder; FreezeString retains an immutable input string. |
| Clone | Make independent native storage. It does not recursively copy arbitrary fields. |
| Protected storage | Frozen and Local records, Local projections, and containers that own them. Frozen pointers, Frozen interfaces, and frozen collection handles occupy ordinary replaceable slots; their selected Frozen data stay immutable. Fields inside protected owners keep the owner's write and address rules. |
| Checked package | A package selected by the [package boundary](rules.md#package-boundary). |
| Service | A struct that embeds `supervise.Service[A]`. Init sets it up; Run does background work. |
| Service method | A declared or promoted method on a service. Exported methods have the [result signature rule](service-results.md). |
| Opaque handle | UncheckedValue, or a named Frozen struct with one UncheckedValue field. Copies share a target whose access rules belong to its package. |
| Service reference | `supervise.Ref[T]`; a copyable reference to the service created by Start. |
| Service run | One service value's background execution and child group. |
| Group; child | A `supervise.Group` owns services and nested groups and stops them in reverse start order. Each is a child of its parent group. |
| Allowed form; matcher | A documented source shape; the checker code that validates its syntax, resolved types, and context. |
