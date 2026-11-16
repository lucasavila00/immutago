package supervise

import (
	"context"
	"testing"
	"testing/synctest"
	"time"
)

// TestInitRequestTimeoutAfterPeerFailure bounds both waits in a setup request.
func TestInitRequestTimeoutAfterPeerFailure(t *testing.T) {
	for _, wait := range []string{"send", "reply"} {
		t.Run(wait, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { checkInitRequestTimeout(t, wait) })
		})
	}
}

// checkInitRequestTimeout keeps Init and its child cleanup ahead of the terminal action.
func checkInitRequestTimeout(t *testing.T, wait string) {
	t.Helper()
	reported, exited := make(chan *Failure, 1), make(chan any, 1)
	top := root(func(value any) { exited <- value }, func(value any, _ []byte) {
		reported <- value.(*Failure)
	})
	requests := make(chan chan string)
	ready := make(chan struct{})
	_, err := Start[initializing](top, &initHooks{
		init: func(*initializing) error { return nil },
		run: func(*initializing, context.Context) {
			<-ready
			if wait == "reply" {
				<-requests
			}
			panic("store failed")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	cleaning, release := make(chan struct{}), make(chan struct{})
	started := make(chan error, 1)
	go func() {
		ref, err := Start[initializing](top, &initHooks{
			init: func(p *initializing) error {
				_, err := Start[initializing](p.UncheckedChildren(), &initHooks{
					init: func(*initializing) error { return nil },
					run: func(_ *initializing, ctx context.Context) {
						if ctx.Err() == nil {
							t.Error("staged child ran with a live context")
						}
						close(cleaning)
						<-release
					},
				})
				if err != nil {
					return err
				}
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				reply := make(chan string, 1)
				close(ready)
				select {
				case requests <- reply:
				case <-ctx.Done():
					return ctx.Err()
				}
				select {
				case <-reply:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
			run: func(*initializing, context.Context) { t.Error("failed Init reached Run") },
		})
		if !ref.IsZero() {
			t.Error("failed Init returned a service reference")
		}
		started <- err
	}()
	failure := <-reported
	synctest.Wait()
	if len(exited) != 0 || len(started) != 0 {
		t.Error("terminal action or Start completed before the request deadline")
	}
	<-cleaning
	synctest.Wait()
	if len(exited) != 0 || len(started) != 0 {
		t.Error("terminal action or Start completed before child cleanup")
	}
	close(release)
	if err := <-started; err != context.DeadlineExceeded {
		t.Fatalf("Init error = %v; want context.DeadlineExceeded", err)
	}
	<-top.Stop()
	if value := <-exited; value != failure || failure.Value != "store failed" {
		t.Fatalf("terminal failure = %v; report = %v", value, failure)
	}
}
