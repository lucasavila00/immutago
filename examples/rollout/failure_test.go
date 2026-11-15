package rollout

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

// TestRetryExhaustion checks the real terminal panic in a separate process with the real retry policy.
func TestRetryExhaustion(t *testing.T) {
	if os.Getenv("IMMUTAGO_ROLLOUT_FAILURE_TEST") == "1" {
		files := sampleFiles(t)
		files.set("rollout.json", candidateConfig)
		synctest.Test(t, func(t *testing.T) {
			_, _ = startTree(t, files, &eventLog{output: os.Stderr})
			files.fail("v1-a.ndjson", -1, false)
			time.Sleep(time.Minute)
			t.Fatal("exhausted reader did not terminate the process")
		})
		return
	}
	command := exec.Command(os.Args[0], "-test.run=^TestRetryExhaustion$", "-test.count=1")
	command.Env = append(os.Environ(), "IMMUTAGO_ROLLOUT_FAILURE_TEST=1")
	output, err := command.CombinedOutput()
	text := string(output)
	if err == nil || strings.Count(text, "attempt failed:") != 6 || strings.Count(text, "panic: supervise:") != 1 {
		t.Fatalf("terminal failure = %v\n%s", err, text)
	}
	fatal := strings.Index(text, "panic: supervise:")
	for _, line := range []string{"reader stopped: v1-b.ndjson", "reader stopped: v2-a.ndjson",
		"reader stopped: v2-b.ndjson", "deployment stopped: checkout-v1", "deployment stopped: checkout-v2",
		"rollout stopped", "policy stopped"} {
		if at := strings.Index(text, line); at < 0 || at > fatal {
			t.Fatalf("resource did not stop before terminal failure: %s\n%s", line, text)
		}
	}
	for _, line := range []string{"rollout started", "policy started", "deployment started: checkout-v1"} {
		if strings.Count(text, line) != 1 {
			t.Fatalf("parent added another retry budget: %s\n%s", line, text)
		}
	}
}
