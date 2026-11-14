package process

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// IntDependency is an exact alias for a dependency.
type IntDependency = proc.UncheckedDependency[*int]

// AliasedArgs uses an alias in a named dependency field.
type AliasedArgs struct {
	im.Shared
	Value IntDependency
}

// MakeAliasedArgs passes a dependency through an ordinary function.
func MakeAliasedArgs(value IntDependency) AliasedArgs { return AliasedArgs{Value: value} }

// ZeroDependency permits the zero dependency and its empty literal.
func ZeroDependency() IntDependency {
	var value IntDependency
	value = IntDependency{}
	return value
}

// TrustedPoint is an ordinary immutable result type.
type TrustedPoint struct {
	im.Shared
	X int
}

// OpaqueArgs stores a trusted callback without inspecting its signature.
type OpaqueArgs struct {
	im.Shared
	Point proc.UncheckedDependency[*TrustedPoint]
	Make  proc.UncheckedDependency[func() *TrustedPoint]
}

// makeTrustedPoint returns a fresh immutable point.
func makeTrustedPoint() *TrustedPoint { return &TrustedPoint{X: 1} }

// MakeOpaqueArgs wraps values whose types mention immutable storage.
func MakeOpaqueArgs() OpaqueArgs {
	return OpaqueArgs{Point: proc.Trust(makeTrustedPoint()), Make: proc.Trust(makeTrustedPoint)}
}
