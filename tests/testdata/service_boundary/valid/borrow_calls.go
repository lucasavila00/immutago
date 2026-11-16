package valid

import "example.com/consumer/api"

type promoted struct{ api.Handle }

// recoverPointer still uses the direct Ref accessor after the assertion.
func recoverPointer(value any) *api.Counter { return value.(api.Handle).Borrow() }

// promotedCall uses the promoted Borrow method.
func promotedCall(ref promoted) { ref.Borrow().Count() }

// expressionCall returns the pointer through a method expression.
func expressionCall(ref api.Handle) *api.Counter { return api.Handle.Borrow(ref) }

// savedMethod keeps a bound service method.
func savedMethod(ref api.Handle) func() int { return ref.Borrow().Count }
