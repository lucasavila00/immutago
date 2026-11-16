package supervise

import (
	"context"
	"testing"
	"testing/synctest"
)

// TestConcurrentInitShutdown reports every accepted failure in reverse registration order.
func TestConcurrentInitShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reports, exited := make(chan *Failure, 2), make(chan any, 1)
		top := root(func(value any) { exited <- value }, func(value any, _ []byte) {
			reports <- value.(*Failure)
		})
		ready := [2]chan struct{}{make(chan struct{}), make(chan struct{})}
		release := [2]chan struct{}{make(chan struct{}), make(chan struct{})}
		started := [2]chan error{make(chan error, 1), make(chan error, 1)}
		for index := range 2 {
			go func() {
				_, err := Start[initializing](top, &initHooks{
					init: func(*initializing) error {
						close(ready[index])
						<-release[index]
						return nil
					},
					run: func(_ *initializing, ctx context.Context) {
						if ctx.Err() == nil {
							t.Error("Run was admitted after Stop closed admission")
						}
						panic(index)
					},
				})
				started[index] <- err
			}()
		}
		<-ready[0]
		<-ready[1]
		stopped := top.Stop()
		close(release[1])
		if err := <-started[1]; err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if len(reports) != 0 {
			t.Error("Stop began Run cleanup while another accepted Init was running")
		}
		close(release[0])
		if err := <-started[0]; err != nil {
			t.Fatal(err)
		}
		<-stopped
		if len(reports) != 2 {
			t.Fatalf("failure reports = %d; want 2", len(reports))
		}
		first, second := <-reports, <-reports
		if first.Value != 0 || second.Value != 1 || <-exited != first {
			t.Fatalf("failure order = %v, %v; want 0 then 1", first, second)
		}
	})
}
