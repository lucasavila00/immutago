package rollout

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
)

// TestLiveComparison checks the supplied logs, partial appends, invalid rows, and HTTP output.
func TestLiveComparison(t *testing.T) {
	files := sampleFiles(t)
	synctest.Test(t, func(t *testing.T) {
		root, ref := startTree(t, files, new(eventLog))
		defer func() { <-root.Stop() }()
		first := reportResponse(ref.Borrow().Read())
		if first.Primary.Requests != 10 || first.Primary.P95Millis != 100 || first.Verdict != "no_candidate" {
			t.Fatal(first)
		}
		files.set("rollout.json", candidateConfig)
		advance()
		if got := reportResponse(ref.Borrow().Read()); got.Candidate.Requests != 6 || got.Verdict != "waiting" {
			t.Fatal(got)
		}
		files.append("v2-a.ndjson", strings.Repeat("{\"status\":200,\"duration_ms\":30}\n", 3))
		files.append("v2-b.ndjson", `{"status":200,"duration_ms":`)
		advance()
		if got := reportResponse(ref.Borrow().Read()); got.Candidate.Requests != 9 {
			t.Fatal(got)
		}
		files.append("v2-b.ndjson", "35}\n")
		advance()
		recorder := httptest.NewRecorder()
		HTTPHandler(ref).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/report", nil))
		var response Response
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if recorder.Code != 200 || response.Verdict != "pass" || response.Candidate.P95Millis != 50 {
			t.Fatal(recorder.Code, response)
		}
		files.append("v2-a.ndjson", "invalid JSON\n")
		advance()
		if got := reportResponse(ref.Borrow().Read()); got.Verdict != "invalid_logs" || got.Candidate.Invalid != 1 {
			t.Fatal(got)
		}
	})
}

// TestPolicyRefreshChangesVerdict keeps all process instances while the simple sibling updates limits.
func TestPolicyRefreshChangesVerdict(t *testing.T) {
	files := sampleFiles(t)
	files.set("rollout.json", candidateConfig)
	files.append("v2-a.ndjson", strings.Repeat("{\"status\":200,\"duration_ms\":200}\n", 4))
	synctest.Test(t, func(t *testing.T) {
		events := new(eventLog)
		root, ref := startTree(t, files, events)
		defer func() { <-root.Stop() }()
		if got := reportResponse(ref.Borrow().Read()); got.Verdict != "fail" {
			t.Fatal(got)
		}
		files.set("policy.json", `{"version":"wide","min_requests":10,
 "max_error_increase_percent":1,"max_p95_increase_ms":200}`)
		advance()
		if got := reportResponse(ref.Borrow().Read()); got.Verdict != "pass" || got.PolicyVersion != "wide" {
			t.Fatal(got)
		}
		files.set("policy.json", "{")
		advance()
		if got := reportResponse(ref.Borrow().Read()); got.Verdict != "pass" || got.PolicyProblem == "" {
			t.Fatal(got)
		}
		files.set("policy.json", `{"version":"strict","min_requests":10,
 "max_error_increase_percent":1,"max_p95_increase_ms":50}`)
		advance()
		if got := reportResponse(ref.Borrow().Read()); got.Verdict != "fail" || got.PolicyProblem != "" {
			t.Fatal(got)
		}
		for _, line := range []string{"policy started", "rollout started", "deployment started: checkout-v1",
			"deployment started: checkout-v2", "reader started: v1-a.ndjson", "reader started: v2-a.ndjson"} {
			if events.count(line) != 1 {
				t.Fatalf("configuration restarted a service: %s", line)
			}
		}
	})
}

// TestFailedSelectionKeepsCurrent checks parse errors and cleanup after partial child setup.
func TestFailedSelectionKeepsCurrent(t *testing.T) {
	files := sampleFiles(t)
	synctest.Test(t, func(t *testing.T) {
		root, ref := startTree(t, files, new(eventLog))
		defer func() { <-root.Stop() }()
		for _, edit := range []string{"{", `{"primary":{"version":"bad","logs":["v2-a.ndjson","missing"]}}`} {
			files.set("rollout.json", edit)
			advance()
			if got := reportResponse(ref.Borrow().Read()); got.Primary.Version != "checkout-v1" || got.Problem == "" {
				t.Fatal(got)
			}
			if active, _ := files.counts("v2-a.ndjson"); active != 0 {
				t.Fatal("failed staged reader retained a file")
			}
		}
		files.set("rollout.json", candidateConfig)
		advance()
		if got := reportResponse(ref.Borrow().Read()); got.Problem != "" || got.Candidate == nil {
			t.Fatal(got)
		}
	})
}

// TestGateChecksErrorIncrease distinguishes error, sample, and latency limits.
func TestGateChecksErrorIncrease(t *testing.T) {
	primary := Totals{Requests: 100, Failures: 1, Buckets: [8]uint64{100}, MaxMillis: 10}
	candidate := primary
	rules := Limits{MinRequests: 100, MaxErrorIncreasePercent: 1, MaxP95IncreaseMillis: 10}
	candidate.Failures = 2
	if verdict, _ := evaluate(primary, candidate, rules); verdict != "pass" {
		t.Fatal(verdict)
	}
	candidate.Failures = 3
	if verdict, reason := evaluate(primary, candidate, rules); verdict != "fail" || !strings.Contains(reason, "error") {
		t.Fatal(verdict, reason)
	}
}
