package supervise

import (
	"context"
	"testing"
)

// TestInvalidGroup rejects invalid public parents before Init can acquire resources.
func TestInvalidGroup(t *testing.T) {
	for _, parent := range []*Group{nil, {}} {
		inits := 0
		hooks := &initHooks{init: func(*initializing) error { inits++; return nil }}
		expectPanic(t, "supervise: nil or zero Group", func() { _, _ = Start[initializing](parent, hooks) })
		expectPanic(t, "supervise: nil or zero Group", func() { StartGroup(parent) })
		expectPanic(t, "supervise: nil or zero Group", func() { parent.Stop() })
		if inits != 0 {
			t.Fatal("Init ran with an invalid parent")
		}
	}
}

// TestChildrenRequiresStart rejects unbound services before child Init can acquire resources.
func TestChildrenRequiresStart(t *testing.T) {
	for _, test := range []struct {
		name     string
		children func() *Group
	}{
		{"Service", new(Service[NoArgs]).Children},
		{"embedded", new(serviceBase).Children},
		{"nested", new(struct{ serviceBase }).Children},
	} {
		t.Run(test.name, func(t *testing.T) {
			expectPanic(t, "supervise: nil or zero Group", func() { test.children() })
			inits := 0
			hooks := &initHooks{init: func(*initializing) error { inits++; return nil }}
			expectPanic(t, "supervise: nil or zero Group", func() {
				_, _ = Start[initializing](test.children(), hooks)
			})
			if inits != 0 {
				t.Fatal("Init ran under an unstarted service")
			}
		})
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
		expectPanic(t, "supervise: embed Service or TerminalService by value through each embedded field", start)
	}
}
