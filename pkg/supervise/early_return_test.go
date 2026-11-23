package supervise

import (
	"context"
	"testing"
	"testing/synctest"
	"time"
)

type returnsAtOnce struct {
	Service[NoArgs]
}

// Run returns without waiting for cancellation.
func (*returnsAtOnce) Run(context.Context) {}

type waitsForCancel struct {
	Service[NoArgs]
}

// Run returns when its context ends.
func (*waitsForCancel) Run(ctx context.Context) { <-ctx.Done() }

type closesOnCancel struct {
	Service[chan struct{}]
}

// Run closes its channel when its context ends, then takes one minute of fake time to stop.
func (s *closesOnCancel) Run(ctx context.Context) {
	<-ctx.Done()
	close(s.Args())
	time.Sleep(time.Minute)
}

type returnsOnClose struct {
	Service[chan struct{}]
}

// Run returns when the channel closes, without waiting for its own cancellation.
func (s *returnsOnClose) Run(context.Context) { <-s.Args() }

// TestEarlyReturnIsReported reports a Run that ends on its own and keeps the root running.
func TestEarlyReturnIsReported(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var returned []string
		root := NewRoot(nil, OnEarlyReturn(func(service string) { returned = append(returned, service) }))
		if _, err := Start[returnsAtOnce](root, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if len(returned) != 1 || returned[0] != "supervise.returnsAtOnce" {
			t.Fatalf("early returns = %v; want [supervise.returnsAtOnce]", returned)
		}
		if _, err := Start[waitsForCancel](root, NoArgs{}); err != nil {
			t.Fatalf("root stopped after an early return: %v", err)
		}
		<-root.Stop()
		if len(returned) != 1 {
			t.Errorf("early returns after Stop = %v", returned)
		}
	})
}

// TestEarlyReturnNotReportedOnStop ignores Runs that end because of shutdown.
func TestEarlyReturnNotReportedOnStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var returned []string
		root := NewRootNoPanic(nil, OnEarlyReturn(func(service string) { returned = append(returned, service) }))
		closed := make(chan struct{})
		if _, err := Start[returnsOnClose](root, closed); err != nil {
			t.Fatal(err)
		}
		if _, err := Start[closesOnCancel](root, closed); err != nil {
			t.Fatal(err)
		}
		group := StartSupervisor(root)
		if _, err := Start[waitsForCancel](group, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		<-group.Stop()
		<-root.Stop()
		if len(returned) != 0 {
			t.Errorf("early returns = %v; want none", returned)
		}
	})
}

// TestEarlyReturnWithoutOption keeps the old silent behavior.
func TestEarlyReturnWithoutOption(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(func(value any, _ []byte) { t.Errorf("report %v", value) })
		if _, err := Start[returnsAtOnce](root, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		<-root.Stop()
	})
}
