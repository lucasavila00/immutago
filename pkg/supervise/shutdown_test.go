package supervise_test

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type alwaysFails struct {
	supervise.Service[*atomic.Int32]
}

// Init counts each setup attempt.
func (s *alwaysFails) Init() error { s.Args().Add(1); return nil }

// Run fails each attempt.
func (*alwaysFails) Run(context.Context) { panic("fail") }

type slowToStop struct {
	supervise.Service[supervise.NoArgs]
}

// Run takes one minute of fake time to stop.
func (*slowToStop) Run(ctx context.Context) {
	<-ctx.Done()
	time.Sleep(time.Minute)
}

// TestStopDisablesRestartsAndEscalation holds shutdown open past the restart limit.
func TestStopDisablesRestartsAndEscalation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var inits atomic.Int32
		root := supervise.NewRoot(func(value any, _ []byte) {
			t.Logf("report %v at %s", value, time.Now().UTC().Format(time.TimeOnly))
		})
		if _, err := supervise.Start[alwaysFails](root, &inits); err != nil {
			t.Fatal(err)
		}
		if _, err := supervise.Start[slowToStop](root, supervise.NoArgs{}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Second)
		synctest.Wait()
		before := inits.Load()
		<-root.Stop()
		if n := inits.Load() - before; n != 0 {
			t.Errorf("Init ran %d more times after Stop", n)
		}
	})
}
