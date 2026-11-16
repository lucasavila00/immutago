# Glossary

| Term | Meaning |
| --- | --- |
| Frozen | Immutable stored data with the recursive [Frozen field rule](types.md). Copies can share backing storage. |
| Frozen record | A struct with `im.Frozen`. |
| Copied | Boolean/numeric data, records with `im.Copied`, and unnamed arrays of Copied values. Frozen records are not Copied. |
| Copied record | A struct with `im.Copied`. All data fields copy by value. Ordinary copies can change independently. It has no string or reference fields. |
| Local value | A read projection, a record with `im.Local`, or an unnamed array of Local values. Local variables, Local records, and Local builders can hold it; package variables, ordinary records, and closures cannot. |
| Read result | A type permitted in a Local field or service data result. See the complete [type list](types.md#read-results--exhaustive). |
| Projection | Local access to a Frozen value. Generated getters use the [field-result rules](projections.md#field-results--exhaustive). |
| Cell; cell holder | An `im.Cell[T]` slot; an unmarked struct containing a named slot or holder field. Generated types such as SettingsCell are holders. Holder pointers can be shared; holder values cannot be copied. |
| Builder | A native slice or map checked for exclusive use before Freeze. |
| FrozenString; FrozenSlice; FrozenMap | Stored string, slice, and map types. Copies share immutable backing storage. |
| Freeze | Wrap native storage without a copy. FreezeSlice and FreezeMap consume a builder; FreezeString retains an immutable input string. |
| Clone | Make independent native storage. It does not recursively copy arbitrary fields. |
| Protected storage | Frozen and Local records, Local projections, and containers that own them. Frozen pointers, Frozen interfaces, and frozen collections occupy ordinary replaceable slots; their selected Frozen data stay immutable. Fields inside protected owners keep the owner's write and address rules. |
| Checked package | A package selected by the [package boundary](rules.md#package-boundary). |
| Service | A struct that embeds `supervise.Service[A]`. |
| Service method | A declared or promoted method on a service. Exported methods have the [result signature rule](service-results.md). |
| UncheckedValue | A stored pointer whose access and synchronization rules belong to the package that declares its payload type. |
| Allowed form | A permitted Go expression or statement, with its required types and context. |
