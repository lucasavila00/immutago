# Validation and review regressions

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
The review below found accepted programs in the previous implementation. The
explicit subset now rejects these forms; the examples remain as regressions.

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
local checks for a change and hosted CI for the full suite. Test success is not a claim that every possible source form has been proved.

## Confirmed failures in the previous implementation

The independent review used the source at `0d7e423`, whose tree matches
`009830a`. Both examples below passed the real CLI with exit status zero.
These were implementation defects, not supported exceptions. The explicit
subset rejects user methods and protected select communication forms.

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
The explicit subset rejects all user-defined receiver methods on protected
values. Only the listed methods on exact `im` collection types remain.

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
are modeled. The explicit subset takes the second approach and also rejects
protected channel expressions in select clauses.

## Cost and scope limits

The old reaching-binding solver retained nested pointer-to-owner maps at every
operation point, including dead local bindings. The explicit subset removes that
solver. Each accepted reference now has one fixed owner; backward liveness remains.

A focused same-machine comparison used separate blocks in one function:

```go
{ var v Value; p := &v; _ = p.X; v.X = 1 }
```

| Blocks | Before: peak RSS (MiB) | Fixed owners: peak RSS (MiB) |
| --- | ---: | ---: |
| 50 | 107.5 | 104.0 |
| 100 | 157.0 | 100.2 |
| 200 | 348.2 | 97.8 |

The baseline was `269eaaf`; the changed solver was `8753467`. Both binaries
checked the same generated consumer packages with
`/usr/bin/time -l <binary> ./scale<N>`, where N was 50, 100, or 200. All six checks passed.
These single observations include package loading and are not isolated solver
benchmarks. At 200 blocks, elapsed time was 0.44 seconds before and 0.19 seconds
after. The memory growth from retained possible-owner maps is removed; these
measurements do not establish a bound for all source programs.

The implementation deliberately omits mutable borrows, escaping references,
arbitrary callback effects, and pointer-return provenance. Whole-owner loans
also reject disjoint field writes. Generic marked structs retain constraint
proof machinery even though generic calls with protected types are rejected.
Removing generic marked structs could reduce that machinery, but would change
the supported API. Recursive collection types likewise explain the structural
proof graph; deleting the graph first would remove needed failure propagation.

## Changes selected from the review

The explicit subset rejects user-defined methods on protected receivers,
protected select communication operations, reference copies and reassignment,
protected multiple assignments, and nested Freeze calls. It keeps fixed local
references, backward NLL, named readers, and the listed collection methods.

This removes the forward possible-owner analysis. It does not remove the
structural proof graph, native builder alias checks, or dependency failure facts;
those still enforce required guarantees. The new source-shape matchers must
follow [the reviewer requirements](../reviewer-focus.md).

Removing all stored local references would be a further product change. It was
not selected: `p := &value; read(p)` remains supported.
