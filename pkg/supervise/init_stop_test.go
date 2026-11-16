package supervise

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"testing/synctest"
)

// initResult records a Start return or a panic in its caller.
type initResult struct {
	ref      Ref[initializing]
	err      error
	panic    any
	returned bool
}

// TestStopWaitsForAcceptedInit checks every Init outcome and its staged child cleanup.
func TestStopWaitsForAcceptedInit(t *testing.T) {
	for _, mode := range []string{"success", "error", "panic", "Goexit"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { checkAcceptedInitStop(t, mode) })
		})
	}
}

// checkAcceptedInitStop keeps peer cancellation after accepted Init and child setup.
func checkAcceptedInitStop(t *testing.T, mode string) {
	t.Helper()
	top := NewRoot(nil)
	peerStopped := make(chan struct{})
	_, err := Start[initializing](top, &initHooks{
		init: func(*initializing) error { return nil },
		run:  func(_ *initializing, ctx context.Context) { <-ctx.Done(); close(peerStopped) },
	})
	if err != nil {
		t.Fatal(err)
	}
	ready, release := make(chan struct{}), make(chan struct{})
	results := make(chan initResult, 1)
	var childRuns, parentRuns atomic.Int32
	cause := errors.New("setup failed")
	go func() {
		var result initResult
		defer func() { result.panic = recover(); results <- result }()
		result.ref, result.err = Start[initializing](top, &initHooks{
			init: func(p *initializing) error {
				_, err := Start[initializing](p.UncheckedChildren(), &initHooks{
					init: func(*initializing) error { return nil },
					run:  func(_ *initializing, ctx context.Context) { <-ctx.Done(); childRuns.Add(1) },
				})
				if err != nil {
					return err
				}
				close(ready)
				<-release
				return finishInit(mode, cause)
			},
			run: func(_ *initializing, ctx context.Context) { <-ctx.Done(); parentRuns.Add(1) },
		})
		result.returned = true
	}()
	<-ready
	stopped := top.Stop()
	synctest.Wait()
	select {
	case <-peerStopped:
		t.Error("Stop cancelled a peer before accepted Init completed")
	case <-stopped:
		t.Error("Stop completed before accepted Init")
	default:
	}
	close(release)
	result := <-results
	<-stopped
	checkInitOutcome(t, mode, cause, result)
	wantRuns := int32(0)
	if mode == "success" {
		wantRuns = 1
	}
	if childRuns.Load() != 1 || parentRuns.Load() != wantRuns {
		t.Fatalf("child Run=%d parent Run=%d; want 1 and %d", childRuns.Load(), parentRuns.Load(), wantRuns)
	}
}

// finishInit supplies the selected setup result in the Start caller.
func finishInit(mode string, cause error) error {
	switch mode {
	case "error":
		return cause
	case "panic":
		panic(cause)
	case "Goexit":
		runtime.Goexit()
	}
	return nil
}

// checkInitOutcome checks publication and the original setup outcome.
func checkInitOutcome(t *testing.T, mode string, cause error, result initResult) {
	t.Helper()
	if result.ref.IsZero() != (mode != "success") || result.returned != (mode == "success" || mode == "error") {
		t.Errorf("Init %s: zero=%t returned=%t", mode, result.ref.IsZero(), result.returned)
	}
	if mode == "error" && result.err != cause || mode != "error" && result.err != nil {
		t.Errorf("Init %s returned error %v", mode, result.err)
	}
	if mode == "panic" && result.panic != cause || mode != "panic" && result.panic != nil {
		t.Errorf("Init %s returned panic %v", mode, result.panic)
	}
}
