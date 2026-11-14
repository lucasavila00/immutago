package supervise

import (
	"bytes"
	"context"
	"testing"
	"testing/synctest"
)

// reportingParent starts a child directly or in a nested group.
type reportingParent struct{ Service[NoArgs] }

// Run waits until ctx ends.
func (*reportingParent) Run(ctx context.Context) { <-ctx.Done() }

// StartChild starts one child that panics.
func (p *reportingParent) StartChild(nested bool) {
	if nested {
		if _, err := Start[reportingChild](StartGroup(p.Children()), NoArgs{}); err != nil {
			panic(err)
		}
		return
	}
	if _, err := Start[reportingChild](p.Children(), NoArgs{}); err != nil {
		panic(err)
	}
}

// reportingChild panics when it runs.
type reportingChild struct{ Service[NoArgs] }

// Run panics with a value that the root reporter can identify.
func (*reportingChild) Run(context.Context) { panic("child report") }

// TestServiceChildrenInheritReporter checks groups created before and after the parent starts.
func TestServiceChildrenInheritReporter(t *testing.T) {
	for _, before := range []bool{true, false} {
		for _, nested := range []bool{true, false} {
			synctest.Test(t, func(t *testing.T) {
				reports := make(chan bool, 1)
				parent := &Group{
					stopped: make(chan struct{}),
					report: func(value any, stack []byte) {
						reports <- value == "child report" && bytes.Contains(stack, []byte("reportingChild"))
					},
				}
				ref, _ := Start[reportingParent](parent, NoArgs{})
				if before {
					ref.Current().StartChild(nested)
				}
				go parent.serve(context.Background())
				if !before {
					synctest.Wait()
					ref.Current().StartChild(nested)
				}
				if !<-reports {
					t.Errorf("wrong child report: before=%t, nested=%t", before, nested)
				}
				<-parent.Stop()
			})
		}
	}
}
