package direct

import "example.com/consumer/api"

// recoverPointer cannot return a Local pointer obtained after a Ref assertion.
func recoverPointer(value any) *api.Counter { return value.(api.Handle).Borrow() }

// expressionCall cannot return a Local pointer from a method expression.
func expressionCall(ref api.Handle) *api.Counter { return api.Handle.Borrow(ref) }
