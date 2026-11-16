package supervise

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

// TestTerminalReportBeforeCleanup keeps failure reporting ahead of a blocked shutdown.
func TestTerminalReportBeforeCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cause := errors.New("terminal cause")
		failure := &Failure{Service: "worker", Value: cause, Stack: []byte("original stack")}
		reported, exiting := make(chan struct{}), make(chan struct{})
		release := make(chan struct{})
		var reports atomic.Int32
		top := root(func(any) { close(exiting) }, func(value any, stack []byte) {
			err, _ := value.(error)
			var got *Failure
			if !errors.As(err, &got) || got != failure || !errors.Is(err, cause) || string(stack) != "original stack" {
				t.Error("terminal report lost its error or stack")
			}
			reports.Add(1)
			close(reported)
		})
		_, err := Start[initializing](top, &initHooks{
			init: func(*initializing) error { return nil },
			run:  func(_ *initializing, ctx context.Context) { <-ctx.Done(); <-release },
		})
		if err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		top.fail(failure)
		top.fail(&Failure{Service: "other", Value: "later failure"})
		synctest.Wait()
		select {
		case <-reported:
		default:
			t.Error("terminal report waited for service cleanup")
		}
		select {
		case <-exiting:
			t.Error("terminal exit did not wait for service cleanup")
		default:
		}
		close(release)
		<-top.Stop()
		if reports.Load() != 1 {
			t.Fatalf("terminal reports = %d; want 1", reports.Load())
		}
	})
}

// processFailure supplies a source frame that must appear in the default report.
type processFailure struct{ Service[string] }

// Run fails at a frame that is separate from the runtime's shutdown worker.
func (s *processFailure) Run(context.Context) {
	if s.Args() == "Goexit" {
		runtime.Goexit()
	}
	panic("original panic")
}

// TestTerminalProcessOutput checks original stacks when attempt reports are disabled.
func TestTerminalProcessOutput(t *testing.T) {
	if mode := os.Getenv("IMMUTAGO_TERMINAL_PROCESS"); mode != "" {
		synctest.Test(t, func(t *testing.T) {
			_, err := Start[processFailure](NewRoot(nil), mode)
			if err != nil {
				t.Fatal(err)
			}
			time.Sleep(time.Minute)
		})
		return
	}
	for _, mode := range []string{"panic", "Goexit"} {
		t.Run(mode, func(t *testing.T) {
			command := exec.Command(os.Args[0], "-test.run=^TestTerminalProcessOutput$")
			command.Env = append(os.Environ(), "IMMUTAGO_TERMINAL_PROCESS="+mode)
			output, err := command.CombinedOutput()
			if err == nil {
				t.Fatal("terminal failure did not stop the process")
			}
			for _, want := range []string{"terminal failure", "processFailure).Run", mode} {
				if !strings.Contains(string(output), want) {
					t.Errorf("process output lacks %q:\n%s", want, output)
				}
			}
		})
	}
}
