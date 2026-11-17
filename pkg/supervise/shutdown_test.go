package supervise

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

type shutdownArgs struct {
	inits *atomic.Int32
	fail  <-chan struct{}
}

type alwaysFails struct {
	Service[shutdownArgs]
}

// Init counts its setup call.
func (s *alwaysFails) Init(*Supervisor) error { s.Args().inits.Add(1); return nil }

// Run fails after the test releases it.
func (s *alwaysFails) Run(context.Context) { <-s.Args().fail; panic("fail") }

type slowToStop struct {
	Service[NoArgs]
}

// Run takes one minute of fake time to stop.
func (*slowToStop) Run(ctx context.Context) {
	<-ctx.Done()
	time.Sleep(time.Minute)
}

// TestStopKeepsFailure holds shutdown open while an older service fails.
func TestStopKeepsFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var inits atomic.Int32
		fail := make(chan struct{})
		terminal := make(chan any, 1)
		root := root(func(value any) { terminal <- value }, func(value any, _ []byte) {
			t.Logf("report %v at %s", value, time.Now().UTC().Format(time.TimeOnly))
		})
		if _, err := Start[alwaysFails](root, shutdownArgs{inits: &inits, fail: fail}); err != nil {
			t.Fatal(err)
		}
		if _, err := Start[slowToStop](root, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		done := root.Stop()
		close(fail)
		<-done
		if failure := (<-terminal).(*Failure); failure.Value != "fail" {
			t.Fatalf("terminal failure = %v", failure)
		}
		if n := inits.Load(); n != 1 {
			t.Errorf("Init ran %d times; want 1", n)
		}
	})
}
