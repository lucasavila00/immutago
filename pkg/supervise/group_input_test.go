package supervise

import (
	"context"
	"testing"
)

// TestInvalidGroup rejects invalid public parents before Init can acquire resources.
func TestInvalidGroup(t *testing.T) {
	for _, parent := range []*Supervisor{nil, {}} {
		inits := 0
		hooks := &initHooks{init: func(*initializing) error { inits++; return nil }}
		expectPanic(t, "supervise: nil or zero Supervisor", func() { _, _ = Start[initializing](parent, hooks) })
		expectPanic(t, "supervise: nil or zero Supervisor", func() { StartSupervisor(parent) })
		expectPanic(t, "supervise: nil or zero Supervisor", func() { parent.Stop() })
		if inits != 0 {
			t.Fatal("Init ran with an invalid parent")
		}
	}
}

// pointerService has the unsupported direct pointer embedding.
type pointerService struct{ *Service[NoArgs] }

// Run supplies the background method needed by Start.
func (*pointerService) Run(context.Context) {}

// serviceBase supplies runtime methods through a value field.
type serviceBase struct{ Service[NoArgs] }

// pointerBase has a nil pointer before its promoted bind method can run.
type pointerBase struct{ *serviceBase }

// Run supplies the background method needed by Start.
func (*pointerBase) Run(context.Context) {}

// pointerAlias keeps the unsupported pointer through an alias.
type pointerAlias = *serviceBase

// indirectService has a pointer inside its value embedding path.
type indirectService struct{ pointerAlias }

// Run supplies the background method needed by Start.
func (*indirectService) Run(context.Context) {}

// TestInvalidEmbedding reports the selected private bind path before application Init.
func TestInvalidEmbedding(t *testing.T) {
	parent := NewRoot(nil)
	defer func() { <-parent.Stop() }()
	for _, start := range []func(){
		func() { _, _ = Start[pointerService](parent, NoArgs{}) },
		func() { _, _ = Start[pointerBase](parent, NoArgs{}) },
		func() { _, _ = Start[indirectService](parent, NoArgs{}) },
	} {
		expectPanic(t, "supervise: embed Service by value through each embedded field", start)
	}
}
