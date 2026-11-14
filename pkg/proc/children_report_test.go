package proc

import (
	"bytes"
	"context"
	"testing"
	"testing/synctest"
)

// reportingParent starts a child directly or in a nested group.
type reportingParent struct{ Process[None] }

// Run waits until ctx ends.
func (*reportingParent) Run(ctx context.Context) { <-ctx.Done() }

// StartChild starts one child that panics.
func (p *reportingParent) StartChild(nested bool) {
	if nested {
		Start[reportingChild](StartSupervisor(p.Supervisor()), None{})
		return
	}
	Start[reportingChild](p.Supervisor(), None{})
}

// reportingChild panics when it runs.
type reportingChild struct{ Process[None] }

// Run panics with a value that the root reporter can identify.
func (*reportingChild) Run(context.Context) { panic("child report") }

// TestProcessChildrenInheritReporter checks groups created before and after the parent starts.
func TestProcessChildrenInheritReporter(t *testing.T) {
	for _, before := range []bool{true, false} {
		for _, nested := range []bool{true, false} {
			synctest.Test(t, func(t *testing.T) {
				reports := make(chan bool, 1)
				parent := &Supervisor{
					stopped: make(chan struct{}),
					report: func(value any, stack []byte) {
						reports <- value == "child report" && bytes.Contains(stack, []byte("reportingChild"))
					},
				}
				pid := Start[reportingParent](parent, None{})
				if before {
					pid.Proc().StartChild(nested)
				}
				go parent.serve(context.Background())
				if !before {
					synctest.Wait()
					pid.Proc().StartChild(nested)
				}
				if !<-reports {
					t.Errorf("wrong child report: before=%t, nested=%t", before, nested)
				}
				<-parent.Stop()
			})
		}
	}
}
