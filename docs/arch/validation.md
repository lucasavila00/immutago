# Validation and known gaps

[Architecture index](README.md)

## Trusted boundary

The [analyzer](../../pkg/analyzer/analyzer.go) skips the package at the fixed
`im` library path and packages identified as standard library source.
[Toolchain detection](../../internal/borrow/toolchain.go) uses the active
`go env GOROOT` source directory. The implementation trusts the Go compiler,
runtime, standard library, and `im` storage methods. It does not prove those
implementations. Reflection and unsafe imports in checked application code
are rejected.

An ordinary Go build does not run these checks. Run the analyzer, or configure
`go vet` with its absolute `-vettool` path, for consumer packages. The standard
driver also analyzes dependencies and carries failure facts. Freeze has no
runtime alias check, consumed flag, or element validation. Code outside this
analysis can retain a writable alias to a frozen container.

The intended contract rejects operations that it cannot prove safe. That
policy is not evidence that every source form has been checked correctly.
The review below found accepted programs that break the intended contract.

## Tests and hosted CI

[Structural tests](../../internal/checker/types_test.go) exercise type graphs.
[Flow tests](../../internal/borrow/flow/flow_test.go) compare backward liveness
with an independent fixed-point solver. Analyzer tests cover source-level
accepted and rejected operations. Runtime collection tests check copies,
reads, iteration, and allocation behavior.

[CLI tests](../../tests/cli_test.go) build the shipped command and copy fixtures
into independent consumer modules. They check accepted and rejected packages,
compiler errors, JSON output, and `go vet` integration. Dependency tests also
select only consumers, so hidden dependency failures must reach them through
facts. Each fixture has a diagnostic snapshot. Review snapshot changes; do
not update them just to make a test pass. JSON mode follows the standard
driver behavior: diagnostics can be present with a successful exit status.

[Hosted CI](../../.github/workflows/ci.yml) runs source checks, formatting,
generated documentation checks, a build, and self-analysis. It runs race-enabled
unit and end-to-end tests on Linux, macOS, and Windows. The
[development guide](../contrib/development.md) lists local commands. Use focused
local checks for a change and hosted CI for the full suite. Current test success
does not resolve the failures below.

## Confirmed review failures

The independent review used the source at `0d7e423`, whose tree matches
`009830a`. Both examples below passed the real CLI with exit status zero.
These are implementation defects, not supported exceptions to the contract.

### Promoted pointer method

```go
type Counter int
func (c *Counter) Inc() { *c++ }
type Value struct { im.Marker; Counter }

func example() {
    v := Value{}
    p := &v
    p.Inc()
    println(p.Counter) // prints 1
}
```

[Receiver checks](../../pkg/analyzer/subset_calls.go) test the outer expression
`p`, which contains marked storage, instead of rejecting the promoted scalar
receiver `*Counter`. The callee's scalar write is outside the marked-type
checks. `v.Inc()` also passes while `p` is live. Rejecting promoted pointer
methods on protected owners would be a small restriction that closes this
case; resolving and checking the actual receiver would preserve more methods.
Neither change is implemented by these notes.

### Select receive assignment after freeze

Go evaluates all select channel operands before it selects a case. A receive
assignment to a selected case's left side occurs later. This accepted program
changes storage after freezing it:

```go
package main

import "github.com/lucasavila00/immutago/pkg/im"

var done = make(chan bool)
var observed = make(chan bool)

func observe(f im.FrozenSlice[bool]) {
    <-done
    println("frozen after:", f.At(0))
    observed <- true
}

func watch(f im.FrozenSlice[bool]) <-chan int {
    println("frozen before:", f.At(0))
    go observe(f)
    return nil
}

func main() {
    s := []bool{false}
    ready := make(chan int, 1)
    ready <- 1
    select {
    case _, s[0] = <-ready:
    case <-watch(im.FreezeSlice(s)):
    }
    done <- true
    <-observed
}
```

The output is `frozen before: false`, then `frozen after: true`. Channel
synchronization orders the later read after the write. This example does not
need a data race. The freeze analysis relies on the generic Go CFG's node
placement. Its event lowering does not reconstruct this delayed assignment to
the second receive target. The ownership checks need explicit select effects,
or must reject builder and protected-value receive targets until those effects
are modeled.

## Cost and scope limits

The reaching-binding solver retains nested pointer-to-owner maps at every
operation point. It copies those maps during full fixed-point scans. A local
review measured the complete CLI process on macOS with 50, 100, and 200
separate blocks in one function, each containing:

```go
{ var v Value; p := &v; _ = p.X; v.X = 1 }
```

| Blocks | Peak resident bytes | Wall time |
| --- | ---: | ---: |
| 50 | 116375552 | 0.21 s |
| 100 | 160595968 | 0.25 s |
| 200 | 348225536 | 0.44 s |

The command was `/usr/bin/time -l env GOWORK=off <binary> ./...` in each
consumer module. These are single observations, including package loading,
not isolated solver benchmarks or a formal complexity bound. They identify a
scaling concern even when loans do not overlap. Compact owner identities,
removal of dead bindings, or a worklist solver need measurement before adoption.

The implementation deliberately omits mutable borrows, escaping references,
arbitrary callback effects, and pointer-return provenance. Whole-owner loans
also reject disjoint field writes. Generic marked structs retain constraint
proof machinery even though generic calls with protected types are rejected.
Removing generic marked structs could reduce that machinery, but would change
the supported API. Recursive collection types likewise explain the structural
proof graph; deleting the graph first would remove needed failure propagation.

## Simplification options

These are review recommendations, not implemented rule changes:

1. Reject promoted pointer methods on protected owners, and reject ownership
   operations in select communication clauses. This closes the demonstrated
   gaps by reducing source forms instead of adding more implicit-effect rules.
2. Keep local shared borrows, but remove pointer copying and rebinding. A
   reference then has one fixed owner. The forward possible-owner analysis can
   be removed; backward liveness still permits writes after the last read.
3. For a larger reduction, allow only direct reader calls such as `read(&value)`
   and read-only reader parameters. Remove stored local references. This removes
   local NLL and alias-origin analysis, but also removes useful supported code.

Keep recursive field checks, builder alias checks, and dependency failure facts.
They enforce required guarantees. The graph and map data structures are ordinary
finite dataflow structures; the demonstrated soundness defects come from missing
Go effects, not from using maps. Retaining irrelevant maps causes the measured
memory concern and should be addressed separately.
