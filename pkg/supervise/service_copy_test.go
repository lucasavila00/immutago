package supervise_test

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// copiedWorker has no application writes after Init.
type copiedWorker struct {
	supervise.Service[chan struct{}]
}

// Run signals entry without changing the service value.
func (s *copiedWorker) Run(ctx context.Context) { close(s.Args()); <-ctx.Done() }

// TestServiceCopyAfterStart checks copies without waiting for Run admission.
func TestServiceCopyAfterStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := supervise.NewRoot(nil)
		defer func() { <-root.Stop() }()
		started := make(chan struct{})
		ref, err := supervise.Start[copiedWorker](root, started)
		if err != nil {
			t.Fatal(err)
		}
		original := ref.Borrow()
		copy := *original
		if copy.Args() != started {
			t.Fatal("copy lost its start arguments")
		}
		<-started
		<-root.Stop()
		stopped := *original
		if stopped.Args() != started || ref.Borrow() != original {
			t.Fatal("shutdown changed the arguments or reference")
		}
	})
}
