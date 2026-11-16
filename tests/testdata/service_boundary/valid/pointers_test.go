package valid_test

import (
	"testing"
	"testing/synctest"
	"time"

	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// TestSelectedInstance keeps two calls on one pointer while Borrow moves to the next attempt.
func TestSelectedInstance(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		control := &api.Control{Failure: make(chan struct{})}
		root := supervise.NewRoot(nil)
		defer func() { <-root.Stop() }()
		ref, err := supervise.Start[api.RestartWorker](root, control)
		if err != nil {
			t.Fatal(err)
		}
		selected := ref.Borrow()
		if selected.Version() != 1 {
			t.Fatal("first call used the wrong instance")
		}
		close(control.Failure)
		synctest.Wait()
		time.Sleep(time.Second)
		synctest.Wait()
		current := ref.Borrow()
		if selected == current || selected.Version() != 1 || current.Version() != 2 {
			t.Fatal("the local pointer did not keep its selected instance")
		}
	})
}
