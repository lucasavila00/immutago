package supervise

import (
	"context"
	"testing"
	"testing/synctest"
)

// TestTerminalWaitsForInitChildren keeps a staged child inside terminal shutdown.
func TestTerminalWaitsForInitChildren(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reported, exited := make(chan *Failure, 1), make(chan any, 1)
		top := root(func(value any) { exited <- value }, func(value any, _ []byte) { reported <- value.(*Failure) })
		fail := make(chan struct{})
		_, err := Start[initializing](top, &initHooks{
			init: func(*initializing) error { return nil },
			run:  func(*initializing, context.Context) { <-fail; panic("peer failed") },
		})
		if err != nil {
			t.Fatal(err)
		}
		ready, releaseInit := make(chan struct{}), make(chan struct{})
		cleaning, releaseChild := make(chan struct{}), make(chan struct{})
		started := make(chan error, 1)
		go func() {
			_, err := Start[initializing](top, &initHooks{
				init: func(p *initializing) error {
					_, err := Start[initializing](p.UnsafeChildren(), &initHooks{
						init: func(*initializing) error { return nil },
						run: func(_ *initializing, ctx context.Context) {
							if ctx.Err() == nil {
								t.Error("staged child ran with a live context")
							}
							close(cleaning)
							<-releaseChild
						},
					})
					close(ready)
					<-releaseInit
					return err
				},
			})
			started <- err
		}()
		<-ready
		close(fail)
		failure := <-reported
		synctest.Wait()
		if len(exited) != 0 {
			t.Error("terminal action did not wait for accepted Init")
		}
		close(releaseInit)
		if err := <-started; err != nil {
			t.Fatal(err)
		}
		<-cleaning
		synctest.Wait()
		if len(exited) != 0 {
			t.Error("terminal action did not wait for staged child cleanup")
		}
		close(releaseChild)
		<-top.Stop()
		if value := <-exited; value != failure || failure.Value != "peer failed" {
			t.Fatalf("terminal failure = %v; report = %v", value, failure)
		}
	})
}
