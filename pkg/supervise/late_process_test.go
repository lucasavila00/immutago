package supervise

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// lateProcessService fails after its root has completed shutdown.
type lateProcessService struct{ Service[NoArgs] }

// Run supplies an error that holds terminal panic formatting open.
func (*lateProcessService) Run(context.Context) { panic(lateProcessError{}) }

// lateProcessError allows a wrongly released Start caller to run before process termination.
type lateProcessError struct{}

// Error delays fatal formatting; the reporter does not format this value.
func (lateProcessError) Error() string {
	time.Sleep(100 * time.Millisecond)
	return "late process failure"
}

// TestLateFailureProcess keeps Start blocked until the terminal panic ends the process.
func TestLateFailureProcess(t *testing.T) {
	if os.Getenv("IMMUTAGO_LATE_FAILURE_PROCESS") != "" {
		top := NewRoot(func(any, []byte) {})
		<-top.Stop()
		if _, err := Start[lateProcessService](top, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		os.Exit(23)
	}
	command := exec.Command(os.Args[0], "-test.run=^TestLateFailureProcess$", "-test.timeout=10s")
	command.Env = append(os.Environ(), "IMMUTAGO_LATE_FAILURE_PROCESS=1")
	output, err := command.CombinedOutput()
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 2 {
		t.Fatalf("late failure exit = %v; want terminal panic exit code 2:\n%s", err, output)
	}
	if !strings.Contains(string(output), "panic: supervise: terminal failure") ||
		!strings.Contains(string(output), "late process failure") {
		t.Fatalf("late failure lost its terminal panic:\n%s", output)
	}
}
