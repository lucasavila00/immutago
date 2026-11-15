package direct

import "example.com/consumer/api"

type promoted struct{ api.Handle }

// recoverPointer still uses the direct Ref accessor after the assertion.
func recoverPointer(value any) *api.Counter { return value.(api.Handle).Borrow() }

// promotedCall must select the embedded Ref field before the direct call.
func promotedCall(ref promoted) { ref.Borrow().Count() }

// expressionCall cannot use the accessor as a method expression.
func expressionCall(ref api.Handle) *api.Counter { return api.Handle.Borrow(ref) }
