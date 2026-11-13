# Mutable parameters: HOW

## Approach

Recognize `Mut` on compiler-resolved formal pointer parameters. Record it as part
of the checked function contract, including across packages. Calls use the
callee's declaration, not the caller's variable spelling.

At a direct call, evaluate arguments in Go order. Resolve the mutable argument
to its whole owned local. Reject the call if any overlapping borrow is live or
another argument borrows the same owner. Grant exclusive access for the call;
restore ordinary ownership when it returns.

Inside the callee, allow writes through that parameter. Keep the reference
non-escaping. Any supported shared reborrow must end before the next write.
Unknown effects and unsupported reference operations remain errors.

## Example

`SetX(&point, 1)` receives temporary exclusive access. `SetX(p, 1)` cannot
upgrade an existing shared reference. A local named `pointMut` cannot upgrade
it either.

## Tradeoffs

The suffix is visible ordinary Go syntax with no runtime checks. Renaming the
formal parameter changes the checked API and requires caller validation.

The implementation must define an explicit allowlist for forwarding mutable
parameters, receiver methods, and generic functions before accepting them.
This decision does not grant mutation authority over aliased slice/map storage
or exempt the collection library from proof.
