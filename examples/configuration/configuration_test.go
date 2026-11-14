package configuration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise"
	"github.com/lucasavila00/immutago/pkg/supervise/supervisetest"
)

// TestLiveLimits checks file reload, quota decisions, and rejection of invalid configuration.
func TestLiveLimits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "limits.json")
	writeLimits(t, path, `{"api":2}`)
	root := supervisetest.NewRoot(t)
	loader := startLoader(t, root, path)
	limiter, err := supervise.Start[Limiter](root, LimiterArgs{Loader: loader, Key: "api", Window: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if limiter.Current().Take() != Allowed || limiter.Current().Take() != Allowed {
		t.Fatal("initial quota was not available after Start")
	}
	if limiter.Current().Take() != Limited {
		t.Fatal("third request exceeded the initial quota")
	}
	before := limiter.Current().Usage()
	writeLimits(t, path, `{"api":`)
	waitFor(t, func() bool { return loader.Current().Problem() != "" })
	if loader.Current().Limit("api") != 2 || loader.Current().Version() != 1 {
		t.Fatal("invalid configuration replaced the last valid limits")
	}
	writeLimits(t, path, `{"api":4}`)
	waitFor(t, func() bool { return loader.Current().Limit("api") == 4 })
	if limiter.Current().Take() != Allowed || limiter.Current().Take() != Allowed {
		t.Fatal("reloaded quota was not available in the same window")
	}
	if limiter.Current().Take() != Limited {
		t.Fatal("request exceeded the reloaded quota")
	}
	if before.Used != 2 || before.Limit != 2 {
		t.Fatal("saved usage changed after reload")
	}
}

// TestLimiterWindow checks that background work opens the next quota window.
func TestLimiterWindow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "limits.json")
	writeLimits(t, path, `{"api":1}`)
	root := supervisetest.NewRoot(t)
	loader := startLoader(t, root, path)
	limiter, err := supervise.Start[Limiter](root, LimiterArgs{
		Loader: loader, Key: "api", Window: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if limiter.Current().Take() != Allowed || limiter.Current().Take() != Limited {
		t.Fatal("one request must fill the first window")
	}
	waitFor(t, func() bool { return limiter.Current().Usage().Window > 0 })
	if limiter.Current().Take() != Allowed {
		t.Fatal("the timer did not restore the quota")
	}
}

// TestConcurrentRequests checks that callers cannot spend the same quota twice.
func TestConcurrentRequests(t *testing.T) {
	path := filepath.Join(t.TempDir(), "limits.json")
	writeLimits(t, path, `{"api":7}`)
	root := supervisetest.NewRoot(t)
	loader := startLoader(t, root, path)
	limiter, err := supervise.Start[Limiter](root, LimiterArgs{Loader: loader, Key: "api", Window: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan bool, 40)
	for range 40 {
		go func() { results <- limiter.Current().Take() == Allowed }()
	}
	accepted := 0
	for range 40 {
		if <-results {
			accepted++
		}
	}
	if accepted != 7 {
		t.Fatalf("accepted %d requests; want 7", accepted)
	}
}

// TestStartReservedRetry checks that a failed file read does not consume a reservation.
func TestStartReservedRetry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "limits.json")
	root := supervisetest.NewRoot(t)
	cache := supervise.Reserve[FileCache]()
	err := supervise.StartReserved(root, cache, FileArgs{Path: path, Every: time.Millisecond})
	if err == nil {
		t.Fatal("missing file did not fail startup")
	}
	writeLimits(t, path, `{"api":3}`)
	if err := supervise.StartReserved(root, cache, FileArgs{Path: path, Every: time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	if got := cache.Current().Text(); got != `{"api":3}` {
		t.Fatalf("initial contents = %q", got)
	}
}

// TestInvalidInitialJSON checks that a service is not published with missing configuration.
func TestInvalidInitialJSON(t *testing.T) {
	cases := []struct{ name, text string }{
		{"empty", ""},
		{"malformed", `{"api":`},
		{"null", "null"},
		{"negative", `{"api":-1}`},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "limits.json")
			writeLimits(t, path, item.text)
			root := supervisetest.NewRoot(t)
			cache, err := supervise.Start[FileCache](root, FileArgs{Path: path, Every: time.Millisecond})
			if err != nil {
				t.Fatal(err)
			}
			_, err = supervise.Start[ConfigLoader](root, LoaderArgs{Cache: cache, Every: time.Millisecond})
			if err == nil {
				t.Fatal("invalid JSON limits did not fail startup")
			}
		})
	}
}

// startLoader starts the real file cache before its configuration consumer.
func startLoader(t *testing.T, root *supervise.Group, path string) supervise.Ref[ConfigLoader] {
	t.Helper()
	cache, err := supervise.Start[FileCache](root, FileArgs{Path: path, Every: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	loader, err := supervise.Start[ConfigLoader](root, LoaderArgs{Cache: cache, Every: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	return loader
}

// writeLimits replaces the input file as an operator would during a configuration change.
func writeLimits(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}

// waitFor waits for observable background work, with a bound on test failure time.
func waitFor(t *testing.T, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !done() {
		if time.Now().After(deadline) {
			t.Fatal("background update did not finish")
		}
		time.Sleep(time.Millisecond)
	}
}
