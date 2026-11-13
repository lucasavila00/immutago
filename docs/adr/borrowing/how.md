# Borrowing: HOW

## Fixed owners and NLL

Match each local reference declaration to one owned local or its marked child:

```go
p := &value
read(p)
value.X = 1 // allowed after the last use
```

The declaration records one compiler variable pair: reference and whole owner.
There is no forward possible-owner solver. Copies, reassignment, delayed
initialization, and unknown reference producers are rejected by the source
matcher. Named reader parameters remain read-only and cannot escape.

Trace uses backward to definitions with the rustc-derived liveness algorithm.
A write fails if a reference to its whole owner is live. Writes through shared
references always fail. Branches and loops remain supported.

## Source forms

The [guide](../../guide/immutability.md) lists the complete allowed forms.
Match those forms before treating the program as proved; reject the fallback.
User methods, protected multiple assignments, nested Freeze, and protected
select communication operations have no accepted shape.

Check dependency bodies and propagate failures through package facts, including
when vet suppresses dependency diagnostics. Retain original failure locations.

Keep source attribution and tests for the retained liveness and graph traversal.
