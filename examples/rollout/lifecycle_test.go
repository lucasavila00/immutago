package rollout

import (
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

// TestPromotionPublishesBeforeCleanup preserves an in-flight report and reuses the candidate process.
func TestPromotionPublishesBeforeCleanup(t *testing.T) {
	files := sampleFiles(t)
	files.set("rollout.json", candidateConfig)
	synctest.Test(t, func(t *testing.T) {
		events := new(eventLog)
		root, ref := startTree(t, files, events)
		defer func() { <-root.Stop() }()
		selected, finish := make(chan struct{}), make(chan struct{})
		result := make(chan Response, 1)
		go func() {
			old := ref.Borrow().Read()
			close(selected)
			<-finish
			response := reportResponse(old)
			result <- response
		}()
		<-selected
		files.blockClose("v1-b.ndjson")
		files.set("rollout.json", promotedConfig)
		<-files.entered
		if got := reportResponse(ref.Borrow().Read()); got.Primary.Version != "checkout-v2" || got.Candidate != nil {
			t.Error(got)
		}
		close(finish)
		if got := <-result; got.Primary.Version != "checkout-v1" || got.Primary.Requests != 10 {
			t.Error(got)
		}
		close(files.release)
		synctest.Wait()
		if events.count("deployment started: checkout-v2") != 1 || events.count("reader started: v2-a.ndjson") != 1 {
			t.Fatal("promotion restarted the current candidate")
		}
		for _, path := range []string{"v1-a.ndjson", "v1-b.ndjson"} {
			if active, _ := files.counts(path); active != 0 {
				t.Fatalf("retired resource remains open: %s", path)
			}
		}
		files.append("v2-a.ndjson", "{\"status\":200,\"duration_ms\":20}\n")
		advance()
		if got := reportResponse(ref.Borrow().Read()); got.Primary.Requests != 7 {
			t.Fatal(got)
		}
	})
}

// TestReaderIOFailureReplaysWithoutDoubleCount retries one stream while its peers keep processing.
func TestReaderIOFailureReplaysWithoutDoubleCount(t *testing.T) {
	files := sampleFiles(t)
	files.set("rollout.json", candidateConfig)
	synctest.Test(t, func(t *testing.T) {
		events := new(eventLog)
		root, ref := startTree(t, files, events)
		defer func() { <-root.Stop() }()
		files.fail("v1-a.ndjson", 1, false)
		advance()
		files.append("v1-b.ndjson", "{\"status\":200,\"duration_ms\":20}\n")
		files.append("v2-a.ndjson", "{\"status\":200,\"duration_ms\":20}\n")
		advance()
		if got := reportResponse(ref.Borrow().Read()); got.Primary.Requests != 11 || got.Candidate.Requests != 7 {
			t.Fatal("peers did not progress during retry", got)
		}
		time.Sleep(2 * time.Second)
		synctest.Wait()
		if got := reportResponse(ref.Borrow().Read()); got.Primary.Requests != 11 {
			t.Fatal("replay changed the request count", got)
		}
		if events.count("reader started: v1-a.ndjson") != 2 {
			t.Fatal(events.snapshot())
		}
		for _, line := range []string{"reader started: v1-b.ndjson", "reader started: v2-a.ndjson",
			"deployment started: checkout-v1", "rollout started", "policy started"} {
			if events.count(line) != 1 {
				t.Fatal("unrelated service restarted", line)
			}
		}
	})
}

// TestDeploymentRetryWaitsForReaders holds child cleanup before the parent can rebuild its sum.
func TestDeploymentRetryWaitsForReaders(t *testing.T) {
	files := sampleFiles(t)
	files.set("rollout.json", candidateConfig)
	synctest.Test(t, func(t *testing.T) {
		events := new(eventLog)
		root, ref := startTree(t, files, events)
		defer func() { <-root.Stop() }()
		files.blockClose("v1-b.ndjson")
		events.failOnce("deployment updated: checkout-v1")
		files.append("v1-a.ndjson", "{\"status\":200,\"duration_ms\":20}\n")
		<-files.entered
		time.Sleep(2 * time.Second)
		if events.count("deployment started: checkout-v1") != 1 {
			t.Error("parent restarted before its old reader closed")
		}
		files.append("v2-a.ndjson", "{\"status\":200,\"duration_ms\":20}\n")
		advance()
		if got := reportResponse(ref.Borrow().Read()); got.Candidate.Requests != 7 {
			t.Error("peer deployment did not progress", got)
		}
		close(files.release)
		time.Sleep(2 * time.Second)
		synctest.Wait()
		if events.count("deployment started: checkout-v1") != 2 || events.count("deployment started: checkout-v2") != 1 {
			t.Fatal(events.snapshot())
		}
		for _, path := range []string{"v1-a.ndjson", "v1-b.ndjson"} {
			if active, peak := files.counts(path); active != 1 || peak != 1 {
				t.Fatalf("overlapping old and new readers for %s: active=%d, peak=%d", path, active, peak)
			}
		}
		if got := reportResponse(ref.Borrow().Read()); got.Primary.Requests != 11 {
			t.Fatal(got)
		}
	})
}

// TestRolloutRetryKeepsPolicy replaces all owned deployments without restarting the root sibling.
func TestRolloutRetryKeepsPolicy(t *testing.T) {
	files := sampleFiles(t)
	files.set("rollout.json", candidateConfig)
	synctest.Test(t, func(t *testing.T) {
		events := new(eventLog)
		root, ref := startTree(t, files, events)
		defer func() { <-root.Stop() }()
		files.fail("rollout.json", 1, true)
		time.Sleep(2 * time.Second)
		synctest.Wait()
		if events.count("rollout started") != 2 || events.count("policy started") != 1 {
			t.Fatal(events.snapshot())
		}
		for _, path := range []string{"v1-a.ndjson", "v1-b.ndjson", "v2-a.ndjson", "v2-b.ndjson"} {
			if events.count("reader started: "+path) != 2 {
				t.Fatal("child did not restart with its owner", path)
			}
			if active, peak := files.counts(path); active != 1 || peak != 1 {
				t.Fatalf("old child survived replacement: %s", path)
			}
		}
		if got := reportResponse(ref.Borrow().Read()); got.Primary.Requests != 10 || got.Candidate.Requests != 6 {
			t.Fatal(got)
		}
	})
}

// TestSelectiveRemovalAndStopOrder stops candidate descendants, then checks dependency shutdown order.
func TestSelectiveRemovalAndStopOrder(t *testing.T) {
	files := sampleFiles(t)
	files.set("rollout.json", candidateConfig)
	synctest.Test(t, func(t *testing.T) {
		events := new(eventLog)
		root, ref := startTree(t, files, events)
		files.set("rollout.json", `{"primary":{"version":"checkout-v1","logs":["v1-a.ndjson","v1-b.ndjson"]}}`)
		advance()
		if got := reportResponse(ref.Borrow().Read()); got.Candidate != nil || got.Primary.Requests != 10 {
			t.Error(got)
		}
		for _, path := range []string{"v2-a.ndjson", "v2-b.ndjson"} {
			if active, _ := files.counts(path); active != 0 {
				t.Error("candidate descendant remains open", path)
			}
		}
		files.set("rollout.json", candidateConfig)
		advance()
		before := len(events.snapshot())
		<-root.Stop()
		var stopped []string
		for _, line := range events.snapshot()[before:] {
			if strings.Contains(line, "stopped") {
				stopped = append(stopped, line)
			}
		}
		want := []string{"reader stopped: v2-b.ndjson", "reader stopped: v2-a.ndjson",
			"deployment stopped: checkout-v2", "reader stopped: v1-b.ndjson", "reader stopped: v1-a.ndjson",
			"deployment stopped: checkout-v1", "rollout stopped", "policy stopped"}
		if !slices.Equal(stopped, want) {
			t.Fatalf("stop order = %q, want %q", stopped, want)
		}
	})
}
