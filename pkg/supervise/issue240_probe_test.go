package supervise

import (
	"context"
	"runtime"
	"testing"
	"testing/synctest"
)

// TestIssue240Probe holds a child failure until its parent starts child cleanup.
func TestIssue240Probe(t *testing.T) {
	for _, where := range []string{"defer", "report"} {
		for _, cause := range []string{"panic", "Goexit"} {
			t.Run(where+"/"+cause, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					notified, held, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
					exited := make(chan any, 1)
					var report func(any, []byte)
					if where == "report" {
						report = func(value any, _ []byte) {
							if _, terminal := value.(*Failure); !terminal {
								close(held)
								<-release
							}
						}
					}
					top := root(func(value any) { exited <- value }, report)
					var children *Group
					parent, err := Start[initializing](top, &initHooks{
						init: func(p *initializing) error {
							children = p.UnsafeChildren()
							_, err := Start[initializing](children, &initHooks{
								init: func(*initializing) error { return nil },
								run: func(*initializing, context.Context) {
									defer func() {
										close(notified)
										if where == "defer" {
											close(held)
											<-release
										}
									}()
									if cause == "Goexit" {
										runtime.Goexit()
									}
									panic("child failure")
								},
							})
							return err
						},
						run: func(*initializing, context.Context) { <-notified },
					})
					if err != nil {
						t.Fatal(err)
					}
					<-held
					synctest.Wait()
					if !parent.Borrow().ended || children.policy.Err() == nil {
						t.Fatal("parent did not start child cleanup")
					}
					close(release)
					synctest.Wait()
					select {
					case <-exited:
					default:
						t.Error("child failure was lost after normal parent return")
					}
					<-top.Stop()
				})
			})
		}
	}
}
