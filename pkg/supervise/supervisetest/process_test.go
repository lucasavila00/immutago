package supervisetest

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// processService fails after it creates a child that must finish cleanup.
type processService struct{ supervise.Service[string] }

// Init starts a child in its child group.
func (s *processService) Init() error {
	_, err := supervise.Start[processChild](s.UnsafeChildren(), supervise.NoArgs{})
	return err
}

// Run supplies the original frame in both failure cases.
func (s *processService) Run(context.Context) {
	if s.Args() == "Goexit" {
		runtime.Goexit()
	}
	panic("original service panic")
}

// processChild records completion after its owner requests shutdown.
type processChild struct {
	supervise.Service[supervise.NoArgs]
}

// Run stops when its parent ends.
func (*processChild) Run(ctx context.Context) {
	<-ctx.Done()
	fmt.Println("child cleanup complete")
}

// TestFailureProcessService is the failing test in the child process.
func TestFailureProcessService(t *testing.T) {
	mode := os.Getenv("IMMUTAGO_TEST_ROOT_FAILURE")
	if mode == "" {
		return
	}
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(t)
		if _, err := supervise.Start[processService](root, mode); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Minute)
	})
}

// TestFailureStopsTestProcess checks terminal failure and cleanup in a subprocess.
func TestFailureStopsTestProcess(t *testing.T) {
	for mode, value := range map[string]string{
		"panic":  "original service panic",
		"Goexit": "supervise: service called runtime.Goexit",
	} {
		t.Run(mode, func(t *testing.T) {
			checkFailureProcess(t, mode, value)
		})
	}
}

// checkFailureProcess checks one failure value and its test process output.
func checkFailureProcess(t *testing.T, mode, value string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0],
		"-test.run=^TestFailureProcessService$", "-test.timeout=10s")
	command.Env = append(os.Environ(), "IMMUTAGO_TEST_ROOT_FAILURE="+mode)
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("test process did not finish: %v\n%s", ctx.Err(), output)
	}
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 2 {
		t.Fatalf("test process error = %v; want terminal panic exit code 2:\n%s", err, output)
	}
	text := string(output)
	for _, want := range []string{
		"panic: supervise: terminal failure", "processService).Run",
		"child cleanup complete", value,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("test output lacks %q:\n%s", want, output)
		}
	}
	if strings.Count(text, "child cleanup complete") != 1 ||
		strings.Index(text, "child cleanup complete") > strings.Index(text, "panic: supervise: terminal failure") {
		t.Errorf("child cleanup did not finish once before the terminal panic:\n%s", output)
	}
}
