package supervise

import (
	"context"
	"runtime"
	"testing"
	"testing/synctest"
)

// TestReporterStopKeepsFailure permits Stop from the reporter without losing failure.
func TestReporterStopKeepsFailure(t *testing.T) {
	for _, where := range []string{"root", "subtree", "unrelated"} {
		t.Run(where, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				reported := make(chan *Failure, 1)
				exited := make(chan any, 1)
				release := make(chan struct{})
				var stop *Group
				top := root(func(value any) { exited <- value }, func(value any, _ []byte) {
					stop.Stop()
					reported <- value.(*Failure)
					<-release
				})
				parent := StartGroup(top)
				stop = top
				if where == "subtree" {
					stop = parent
				} else if where == "unrelated" {
					stop = StartGroup(top)
				}
				_, err := Start[initializing](parent, &initHooks{
					init: func(*initializing) error { return nil },
					run:  func(*initializing, context.Context) { panic("reported failure") },
				})
				if err != nil {
					t.Fatal(err)
				}
				failure := <-reported
				synctest.Wait()
				select {
				case <-top.Stop():
					t.Error("Stop completed before its reporter returned")
				default:
				}
				close(release)
				<-top.Stop()
				if value := <-exited; value != failure || failure.Value != "reported failure" {
					t.Fatalf("terminal failure = %v, report = %v", value, failure)
				}
			})
		})
	}
}

// TestConcurrentFailureReports keeps the first failure and reports each service once.
func TestConcurrentFailureReports(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reports, exited := make(chan *Failure, 2), make(chan any, 2)
		release := make(chan struct{})
		top := root(func(value any) { exited <- value }, func(value any, _ []byte) {
			reports <- value.(*Failure)
			<-release
		})
		triggers := [2]chan struct{}{make(chan struct{}), make(chan struct{})}
		for index, trigger := range triggers {
			_, err := Start[initializing](top, &initHooks{
				init: func(*initializing) error { return nil },
				run: func(*initializing, context.Context) {
					<-trigger
					if index == 1 {
						runtime.Goexit()
					}
					panic("first failure")
				},
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		synctest.Wait()
		close(triggers[0])
		first := <-reports
		close(triggers[1])
		second := <-reports
		close(release)
		<-top.Stop()
		if value := <-exited; value != first || first.Value != "first failure" || second.Value != errGoexit {
			t.Fatalf("exit = %v, first = %v, second = %v", value, first, second)
		}
		if len(exited) != 0 || len(reports) != 0 {
			t.Fatal("failure was reported or selected more than once")
		}
	})
}

// TestCancelledRunFailure keeps failures after the Run context ends.
func TestCancelledRunFailure(t *testing.T) {
	for _, mode := range []string{"panic", "Goexit"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				exited := make(chan any, 1)
				top := root(func(value any) { exited <- value }, nil)
				_, err := Start[initializing](top, &initHooks{
					init: func(*initializing) error { return nil },
					run: func(_ *initializing, ctx context.Context) {
						<-ctx.Done()
						if mode == "Goexit" {
							runtime.Goexit()
						}
						panic("cleanup failed")
					},
				})
				if err != nil {
					t.Fatal(err)
				}
				synctest.Wait()
				<-top.Stop()
				failure := (<-exited).(*Failure)
				if mode == "Goexit" && failure.Value != errGoexit || mode == "panic" && failure.Value != "cleanup failed" {
					t.Fatalf("terminal failure = %v", failure)
				}
			})
		})
	}
}
