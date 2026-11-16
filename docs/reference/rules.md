# Language reference

This reference defines the complete Go forms that Immutago accepts.
Use the [common forms](common-forms.md) with each type's forms. **Exhaustive**
means all allowed forms are listed. Unlisted operations on protected storage
are rejected. Rejected examples are not exhaustive.

| Reference | Rules |
| --- | --- |
| [Type rules](types.md) | Frozen, Copied, Local, and read-result types. |
| [Common forms](common-forms.md) | Stored types, construction, reads, pointers, calls, and statements. |
| [Frozen records](frozen-records.md) | Immutable record fields and stored pointers. |
| [Frozen interfaces](frozen-interfaces.md) | Interface declarations, implementations, conversions, and calls. |
| [Local](local.md) | Local records, projections, pointers, and builders. |
| [Copied](copied.md) | Value records and independent copies. |
| [Cell](cell.md) | Publication slots and their holders. |
| [Generated projections](projections.md) | Generated names, field results, and collection reads. |
| [Projection primitives](projection-primitives.md) | Fixed projection APIs. |
| [Frozen collections](frozen.md) | Strings, slices, maps, and builders. |
| [UncheckedValue](unchecked-values.md) | Stored pointers controlled by their declaring package. |
| [Service declarations and results](service-results.md) | Value embedding, stored arguments, and exported result types. |
| [Glossary](glossary.md) | Terms used in this reference. |

## Required generation

Frozen records and Frozen interfaces require named declarations at package scope
in non-test Go files. A named Frozen record with one named UncheckedValue field
and no other data fields is exempt: it can be generic, local, or in a test file,
and needs no generated code. Local and Copied records also need no generation.

Every package with declarations that require generation needs this exact directive:

```go
//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen
```

| Package setup — exhaustive | Requirement |
| --- | --- |
| Frozen record or Frozen interface that requires generation | Named package-scope type in a `.go` file that does not end in `_test.go`. |
| Directive | Exactly one copy of the command above, at column 1 in a non-test package source file. |
| Output | Keep the current `zz_immutago.gen.go` for the source declarations and build configuration. |
| Frozen records that require generation | Typed field projections, Projection constructors, and Cells. |
| Frozen interfaces | Local method projections and Projection constructors. |
| No declarations that require generation remain | Remove the directive and output, or keep current empty generated output. |

Generation uses the active `GOOS`, `GOARCH`, and build tags. Build constraints can
select declarations; the generated file has no constraint. Generation, checking,
and compilation must use the same configuration.

| Configurations — exhaustive | Required output |
| --- | --- |
| Same generated declarations, field types, array lengths, and interface methods | One file can serve all configurations that generate the same bytes. Constrained files can use its projections and Cells. |
| Different generated output | Current output for each configuration's check and build. |
| A configuration excludes all declarations that require generation | An empty generated package file, or no directive and no output. |
| Imported Frozen types | Current output in their owning packages for the same configuration. |

Generated code keeps the normal type and source rules. Only the exact current
generated file can form interior Frozen-record addresses.
See [generation commands and build configurations](../guides/getting-started.md#check-in-ci).

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `type Settings struct { im.Frozen; Limit int }` with no generation directive | Add the exact directive and generate. |
| That Settings declaration inside a function or `_test.go` file | Move it to a non-test Go file at package scope. |
| An anonymous Frozen record, including a field type | Declare a named record and use that type. |
| An alternate generation command in the directive | Use the command above. |
| Manually changed or stale generated output | Regenerate with the same build configuration. |
| Output made with `GOFLAGS=-tags=audit`, checked with different schema types and no tag | Regenerate without the tag, or use the same declarations and type dependencies in both configurations. |

## Package boundary

The checker checks a package in any of these cases, including its test files:

| Package checks — exhaustive | Example |
| --- | --- |
| Imports `im` or `supervise`, including an alias or blank import | `import _ "github.com/lucasavila00/immutago/pkg/im"` |
| Uses an im or supervise type, including inside another type | `var point api.Point`, where Point embeds `im.Frozen` |
| Uses a function, interface, or generic type that mentions an im or supervise type | `callback := api.Publish`, where Publish takes `*api.Point`; `api.Alias[int]`, where Alias has a Cell type in its constraint |
| Imports a package that failed its checks, directly or through another package | Import a package with an invalid immutable field. |

- Aliases, type arguments, and generic constraints also count.
- A package that uses only supervise receives service type and result checks.
  Using im types also selects the im source checks.
- The CLI and `go vet` use the same rules.
- Other packages are trusted to obey Go contracts, including immutable strings.
- The `im`, `supervise`, and standard-library implementations are trusted.
- The generated Go test runner is skipped; user tests are checked.
- User code has no suppression comments. `im.UncheckedValue` delegates
  payload access and synchronization to its declaring package.
- Importing `unsafe` or `reflect` is allowed; ordinary values use ordinary Go rules.
- Protected object values need a listed form: `reflect.ValueOf(1)` is allowed,
  but `reflect.ValueOf(local)` with a Local record value fails the interface-boxing rule.
  Frozen values, pointers, and their ordinary holders use the listed Go forms.
  Callers must not give a Frozen pointer to an API that changes its target;
  mutation effects through ordinary interfaces, reflection, or unsafe code are not proved.

### Functions without a body — exhaustive

A function without a Go body can use these types in its receiver, parameters,
results, and generic constraints:

| Allowed types | Example |
| --- | --- |
| Ordinary Go types that contain no im type | `func cpuID() uint64`, `func checksum(data []byte) uint64` |
| Copied values, UncheckedValue, and the Frozen records in the [read-result list](types.md#read-results--exhaustive), including their ordinary holders | `func decode(value *Score)` where Score embeds im.Copied |

Unlisted im types require a Go body.

| Rejected examples — non-exhaustive | Use instead |
| --- | --- |
| Cgo in a package that imports or uses im | Ordinary typed Go code. |
| `func read(value *Settings) int` with a generated Frozen Settings record and no body | Supply a Go body. |

An imported package's error includes one cause and a command to see its other errors.
