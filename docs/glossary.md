# Glossary

| Term | Meaning |
| --- | --- |
| Frozen | Immutable stored data with the recursive [Frozen field rule](types.md). Copies can share backing storage. |
| Frozen record | A struct with `im.Frozen`. |
| Copied | Boolean/numeric data, opaque service references, and unnamed arrays of Copied values. Frozen records are not Copied. |
| Local value | A read projection, a record with `im.Local`, or an unnamed array of Local values. It cannot enter persistent storage. |
| Read result | Copied or Local. This is a type category, not a service-specific field rule. |
| Projection | Local access to a Frozen value. Generated types such as SettingsProjection return Copied fields or more projections. |
| Cell; cell holder | An `im.Cell[T]` slot; an unmarked struct containing a named slot or holder field. Generated types such as SettingsCell are holders. |
| Builder | A native slice or map checked for exclusive use before Freeze. |
| FrozenString; FrozenSlice; FrozenMap | Stored string, slice, and map types. Copies share immutable backing storage. |
| Freeze | Wrap native storage without a copy. FreezeSlice and FreezeMap consume a builder; FreezeString retains an immutable input string. |
| Clone | Make independent native storage. It does not recursively copy arbitrary fields. |
| Protected storage | Marked records, frozen collections, projections, and containers that contain them. |
| Checked package | A package selected by the [package boundary](rules.md#package-boundary). |
| Service | A struct that embeds `supervise.Service[A]`. Init sets it up; Run does background work. |
| Service method | A declared or promoted method on a service. Exported methods have the [read-result rule](service-results.md). |
| Service reference | `supervise.Ref[T]`; each Current call selects the current service after any restart. |
| Service run | One service value's background execution and child group. |
| Group; child | A `supervise.Group` manages services and nested groups; each is a child of its parent group. |
| Allowed form; matcher | A documented source shape; the checker code that validates its syntax, resolved types, and context. |
