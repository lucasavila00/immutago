package supervise

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

// TestCompletedRunDuringFailureCleanup keeps normal completion terminal through cleanup.
func TestCompletedRunDuringFailureCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		terminal := make(chan any, 1)
		parent := root(func(value any) { terminal <- value }, nil)
		group := StartGroup(parent)
		started, cleanup, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
		var runs atomic.Int32
		_, err := Start[initializing](group, &initHooks{
			init: func(*initializing) error { return nil },
			run: func(p *initializing, _ context.Context) {
				runs.Add(1)
				_, childErr := Start[initializing](p.Children(), &initHooks{
					init: func(*initializing) error { return nil },
					run: func(_ *initializing, ctx context.Context) {
						close(started)
						<-ctx.Done()
						close(cleanup)
						<-release
					},
				})
				if childErr != nil {
					panic(childErr)
				}
				<-started
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		<-cleanup
		_, err = Start[initializing](group, &initHooks{
			init: func(*initializing) error { return nil },
			run:  func(*initializing, context.Context) { panic("sibling") },
		})
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(31 * time.Second)
		synctest.Wait()
		close(release)
		<-terminal
		<-parent.Stop()
		if runs.Load() != 1 || group.children.count != 0 {
			t.Fatalf("completed Run calls = %d, retained children = %d", runs.Load(), group.children.count)
		}
	})
}

// TestGoexitIsTerminal checks that Goexit reports once without an extra retry.
func TestGoexitIsTerminal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		terminal := make(chan any, 1)
		reported := make(chan []byte, 1)
		parent := root(func(value any) { terminal <- value }, func(value any, stack []byte) {
			if value != errGoexit {
				t.Errorf("Goexit report = %v", value)
			}
			reported <- stack
		})
		var runs atomic.Int32
		_, err := Start[initializing](parent, &initHooks{
			init: func(*initializing) error { return nil },
			run:  func(*initializing, context.Context) { runs.Add(1); runtime.Goexit() },
		})
		if err != nil {
			t.Fatal(err)
		}
		failure := (<-terminal).(*Failure)
		stack := <-reported
		<-parent.Stop()
		if failure.Value != errGoexit || runs.Load() != 1 || parent.children.count != 0 {
			t.Fatalf("Goexit failure=%v runs=%d children=%d", failure, runs.Load(), parent.children.count)
		}
		if !bytes.Contains(stack, []byte("runtime.Goexit")) || !bytes.Contains(failure.Stack, []byte("initializing).Run")) {
			t.Fatalf("Goexit lost the original stack:\n%s", stack)
		}
	})
}

// TestStopRejectsLateFailure checks both subtree and root admission after Stop.
func TestStopRejectsLateFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var exits atomic.Int32
		parent := root(func(any) { exits.Add(1) }, nil)
		nested := StartGroup(parent)
		<-nested.Stop()
		nested.fail(&Failure{Service: "stopped subtree", Value: "late failure"})
		<-parent.Stop()
		parent.fail(&Failure{Service: "stopped root", Value: "late failure"})
		synctest.Wait()
		if exits.Load() != 0 {
			t.Fatal("Stop accepted a later terminal failure")
		}
	})
}

// TestTerminalExitPrecedesStopCompletion prevents a caller from exiting before the failure hook.
func TestTerminalExitPrecedesStopCompletion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cleaned, exiting, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
		parent := root(func(any) { close(exiting); <-release }, nil)
		_, err := Start[initializing](parent, &initHooks{
			init: func(*initializing) error { return nil },
			run:  func(_ *initializing, ctx context.Context) { <-ctx.Done(); close(cleaned) },
		})
		if err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		parent.fail(&Failure{Service: "worker", Value: "terminal"})
		<-exiting
		select {
		case <-cleaned:
		default:
			t.Error("terminal exit ran before service cleanup")
		}
		done := parent.Stop()
		select {
		case <-done:
			t.Error("Stop completed before terminal exit")
		default:
		}
		close(release)
		<-done
	})
}

// TestPendingSubtreeStopsWithoutRun checks initial setup at each ownership depth.
func TestPendingSubtreeStopsWithoutRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		top := NewRoot(nil)
		pending := newGroup(top, false)
		nested := StartGroup(pending)
		var inits, runs atomic.Int32
		ref, err := Start[initializing](nested, &initHooks{
			init: func(p *initializing) error { inits.Add(1); p.Children(); return nil },
			run:  func(*initializing, context.Context) { runs.Add(1) },
		})
		if err != nil {
			t.Fatal(err)
		}
		<-pending.Stop()
		pending.activate()
		nested.activate()
		synctest.Wait()
		expectInitEnded(t, ref.Borrow())
		if inits.Load() != 1 || runs.Load() != 0 {
			t.Fatal("stopped pending setup ran or was initialized again")
		}
		<-top.Stop()
	})
}

// TestAncestorStopEndsRestartPolicy checks propagation without early Run cancellation.
func TestAncestorStopEndsRestartPolicy(t *testing.T) {
	for _, where := range []string{"group", "Init", "Run"} {
		t.Run(where, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { checkAncestorStop(t, where) })
		})
	}
}

// checkAncestorStop holds a newer child open while the older subtree stops retrying.
func checkAncestorStop(t *testing.T, where string) {
	t.Helper()
	parent := root(func(value any) { t.Errorf("root escalated: %v", value) }, nil)
	var inits atomic.Int32
	child := &initHooks{
		init: func(*initializing) error { inits.Add(1); return nil },
		run:  func(*initializing, context.Context) { panic("child failure") },
	}
	ended := make(chan struct{})
	if where == "group" {
		nested := StartGroup(parent)
		_, _ = Start[initializing](nested, child)
		_, _ = Start[initializing](nested, &initHooks{
			init: func(*initializing) error { return nil },
			run:  func(_ *initializing, ctx context.Context) { <-ctx.Done(); close(ended) },
		})
	} else {
		_, _ = Start[initializing](parent, &initHooks{
			init: func(p *initializing) error {
				if where == "Init" {
					_, _ = Start[initializing](p.Children(), child)
				}
				return nil
			},
			run: func(p *initializing, ctx context.Context) {
				if where == "Run" {
					_, _ = Start[initializing](p.Children(), child)
				}
				<-ctx.Done()
				close(ended)
			},
		})
	}
	cleanup, release := make(chan struct{}), make(chan struct{})
	_, _ = Start[initializing](parent, &initHooks{
		init: func(*initializing) error { return nil },
		run: func(_ *initializing, ctx context.Context) {
			<-ctx.Done()
			close(cleanup)
			<-release
		},
	})
	time.Sleep(2 * time.Second)
	before := inits.Load()
	done := parent.Stop()
	<-cleanup
	time.Sleep(time.Minute)
	synctest.Wait()
	if got := inits.Load(); got != before {
		t.Errorf("Init count = %d, want %d", got, before)
	}
	select {
	case <-ended:
		t.Error("older Run context ended before newer child stopped")
	default:
	}
	close(release)
	<-done
}

// TestStopDuringAdmittedRestartInit lets Init finish without publishing its candidate.
func TestStopDuringAdmittedRestartInit(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "error"}[fail], func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				parent := NewRoot(nil)
				entered := make(chan *initializing, 1)
				initRelease, slowRelease, cleanup := make(chan struct{}), make(chan struct{}), make(chan struct{})
				var inits, runs atomic.Int32
				ref, _ := Start[initializing](parent, &initHooks{
					init: func(p *initializing) error {
						if inits.Add(1) > 1 {
							p.Children()
							entered <- p
							<-initRelease
							if fail {
								return errors.New("setup failure")
							}
						}
						return nil
					},
					run: func(*initializing, context.Context) { runs.Add(1); panic("retry") },
				})
				original := ref.Borrow()
				candidate := <-entered
				_, _ = Start[initializing](parent, &initHooks{
					init: func(*initializing) error { return nil },
					run: func(_ *initializing, ctx context.Context) {
						<-ctx.Done()
						close(cleanup)
						<-slowRelease
					},
				})
				synctest.Wait()
				done := parent.Stop()
				<-cleanup
				close(initRelease)
				synctest.Wait()
				expectInitEnded(t, candidate)
				if ref.Borrow() != original || runs.Load() != 1 || inits.Load() != 2 {
					t.Errorf("restart after stop: changed=%t, runs=%d, Init calls=%d",
						ref.Borrow() != original, runs.Load(), inits.Load())
				}
				close(slowRelease)
				<-done
			})
		})
	}
}
