package supervise_test

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type shutdownArgs struct {
	inits *atomic.Int32
	fail  <-chan struct{}
}

type alwaysFails struct {
	supervise.Service[shutdownArgs]
}

// Init counts each setup attempt.
func (s *alwaysFails) Init() error { s.Args().inits.Add(1); return nil }

// Run fails each attempt.
func (s *alwaysFails) Run(context.Context) { <-s.Args().fail; panic("fail") }

type slowToStop struct {
	supervise.Service[supervise.NoArgs]
}

// Run takes one minute of fake time to stop.
func (*slowToStop) Run(ctx context.Context) {
	<-ctx.Done()
	time.Sleep(time.Minute)
}

// TestStopSuppressesFailure holds shutdown open while an older service fails.
func TestStopSuppressesFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var inits atomic.Int32
		fail := make(chan struct{})
		root := supervise.NewRoot(func(value any, _ []byte) {
			t.Logf("report %v at %s", value, time.Now().UTC().Format(time.TimeOnly))
		})
		if _, err := supervise.Start[alwaysFails](root, shutdownArgs{inits: &inits, fail: fail}); err != nil {
			t.Fatal(err)
		}
		if _, err := supervise.Start[slowToStop](root, supervise.NoArgs{}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		done := root.Stop()
		close(fail)
		<-done
		if n := inits.Load(); n != 1 {
			t.Errorf("Init ran %d times; want 1", n)
		}
	})
}
