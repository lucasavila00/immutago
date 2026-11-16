package trading

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

// TestRetryExhaustion checks the real runtime retry limit and final cleanup in a subprocess.
func TestRetryExhaustion(t *testing.T) {
	if os.Getenv("IMMUTAGO_TRADING_FAILURE_TEST") == "1" {
		files := sampleFiles(t)
		files.set("trading.json", candidateConfig)
		synctest.Test(t, func(t *testing.T) {
			venue := newFakeVenue()
			venue.subscribeFailures["desk-a/AAA"] = -1
			_ = startBot(t, files, venue, &eventLog{output: os.Stderr})
			time.Sleep(time.Minute)
			t.Fatal("exhausted worker did not terminate the process")
		})
		return
	}
	command := exec.Command(os.Args[0], "-test.run=^TestRetryExhaustion$", "-test.count=1")
	command.Env = append(os.Environ(), "IMMUTAGO_TRADING_FAILURE_TEST=1")
	output, err := command.CombinedOutput()
	text := string(output)
	if err == nil || strings.Count(text, "attempt failed:") != 6 || strings.Count(text, "panic: supervise:") != 1 {
		t.Fatalf("terminal failure = %v\n%s", err, text)
	}
	fatal := strings.Index(text, "panic: supervise:")
	for _, line := range []string{"maker stopped: wide/BBB", "maker stopped: tight/AAA", "maker stopped: tight/BBB",
		"strategy stopped: wide", "strategy stopped: tight", "trading stopped", "policy stopped"} {
		if at := strings.Index(text, line); at < 0 || at > fatal {
			t.Fatalf("resource did not stop before terminal failure: %s\n%s", line, text)
		}
	}
	for _, line := range []string{"trading started", "policy started", "strategy started: wide"} {
		if strings.Count(text, line) != 1 {
			t.Fatalf("parent added another retry budget: %s\n%s", line, text)
		}
	}
}
