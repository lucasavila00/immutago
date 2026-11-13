# Mutable parameters: WHAT

## Goal

Let a function temporarily mutate an owned value, like a Rust `&mut T`.
Declare that permission explicitly with a `Mut` suffix on the parameter name.
This is a proposed checker extension, not implemented by this documentation branch.

## Example

```go
func SetX(pointMut *Point, x int) {
    pointMut.X = x
}

var point Point
SetX(&point, 1) // allowed: point is owned and has no live borrows

p := &point
SetX(&point, 2) // rejected: p is used below
read(p)
```

## Rules

- A formal `*T` parameter ending in `Mut` declares an exclusive, call-scoped
  mutable borrow. An ordinary `*T` parameter remains read-only.
- The caller supplies an owned local value with no live shared or mutable borrows.
- The callee may write through the parameter. It cannot return, store, or capture
  the reference, or pass it to deferred or asynchronous work.
- Renaming a local variable to end in `Mut` grants no permission.
- The checker verifies both the call and the function body. This is not a trust annotation.

Slice/map backing-storage ownership requires a separate rule. Copying their
handles does not establish independent ownership.
