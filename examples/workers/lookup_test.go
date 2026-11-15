package workers

import (
	"testing"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise/supervisetest"
)

// TestDispatcherLookup checks use of a returned handle and absence of an unknown variant.
func TestDispatcherLookup(t *testing.T) {
	root := supervisetest.NewRoot(t)
	_, dispatcher := startDispatcher(t, root, time.Hour)
	await(t, func() bool {
		dispatcher.Current().Refresh()
		return dispatcher.Current().Version("blue") == 1
	})
	route := dispatcher.Current().Lookup("blue")
	if !route.Available {
		t.Fatal("ready pool was not returned")
	}
	result := route.Pool.Current().Process("Hello")
	if !result.Available || !result.Text.EqualString("HELLO") {
		t.Fatal("returned pool did not process the request")
	}
	missing := dispatcher.Current().Lookup("missing")
	if missing.Available {
		t.Fatal("unknown variant returned a pool")
	}
}
