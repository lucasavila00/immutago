package modelserve

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// TestRetryExhaustion uses a child process because the runtime's terminal panic is intentional.
func TestRetryExhaustion(t *testing.T) {
	if os.Getenv("IMMUTAGO_MODELSERVE_FAILURE_TEST") == "1" {
		synctest.Test(t, func(t *testing.T) {
			files := sampleFiles()
			logger := log.New(os.Stderr, "", 0)
			root := supervise.NewRoot(func(value any, _ []byte) { logger.Printf("attempt failed: %v", value) })
			router, err := NewRouter(root, files, time.Second, logger)
			if err != nil {
				t.Fatal(err)
			}
			if err := router.SetChallenger("challenger.json"); err != nil {
				t.Fatal(err)
			}
			files.fail("baseline.json", -1)
			time.Sleep(time.Minute)
			t.Fatal("retry exhaustion did not terminate the process")
		})
		return
	}
	command := exec.Command(os.Args[0], "-test.run=^TestRetryExhaustion$", "-test.count=1")
	command.Env = append(os.Environ(), "IMMUTAGO_MODELSERVE_FAILURE_TEST=1")
	output, err := command.CombinedOutput()
	text := string(output)
	if err == nil || strings.Count(text, "attempt failed:") != 6 || strings.Count(text, "panic: supervise:") != 1 {
		t.Fatalf("terminal failure = %v\n%s", err, text)
	}
	fatal := strings.Index(text, "panic: supervise:")
	for _, message := range []string{"model stopped: candidate.json", "switcher stopped: challenger.json",
		"switcher stopped: primary.json"} {
		if index := strings.Index(text, message); index < 0 || index > fatal {
			t.Fatalf("missing ordered cleanup %q\n%s", message, text)
		}
	}
	if strings.Count(text, "switcher ready: primary.json") != 1 ||
		strings.Count(text, "switcher ready: challenger.json") != 1 {
		t.Fatalf("a parent added another retry budget:\n%s", text)
	}
}
