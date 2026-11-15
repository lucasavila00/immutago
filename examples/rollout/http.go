package rollout

import (
	"encoding/json"
	"net/http"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// ReleaseResponse contains ordinary JSON data for one release.
type ReleaseResponse struct {
	Version      string  `json:"version"`
	Replicas     int     `json:"replicas"`
	Requests     uint64  `json:"requests"`
	Failures     uint64  `json:"failures"`
	Invalid      uint64  `json:"invalid_records"`
	ErrorPercent float64 `json:"error_percent"`
	P95Millis    int     `json:"p95_ms"`
}

// Response contains one selected report, including any configuration errors.
type Response struct {
	Primary       ReleaseResponse  `json:"primary"`
	Candidate     *ReleaseResponse `json:"candidate"`
	Verdict       string           `json:"verdict"`
	Reason        string           `json:"reason"`
	Problem       string           `json:"problem,omitempty"`
	PolicyVersion string           `json:"policy_version"`
	PolicyProblem string           `json:"policy_problem,omitempty"`
}

// HTTPHandler serves complete reports; all service lifetime changes belong to Rollout.
func HTTPHandler(ref supervise.Ref[Rollout]) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /report", func(w http.ResponseWriter, _ *http.Request) {
		selected := ref.Borrow().Read()
		response := reportResponse(selected)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	return mux
}

// reportResponse finishes all reads from the request's selected immutable report.
func reportResponse(selected ReportProjection) Response {
	response := Response{
		Primary: releaseResponse(selected.Primary()), Verdict: selected.Verdict().Unsafe(),
		Reason: selected.Reason().Unsafe(), Problem: selected.Problem().Unsafe(),
		PolicyVersion: selected.PolicyVersion().Unsafe(), PolicyProblem: selected.PolicyProblem().Unsafe(),
	}
	if candidate := selected.Candidate(); !candidate.IsNil() {
		value := releaseResponse(candidate)
		response.Candidate = &value
	}
	return response
}

// releaseResponse calculates presentation values from one release's copied totals.
func releaseResponse(selected ReleaseProjection) ReleaseResponse {
	totals := selected.Totals()
	return ReleaseResponse{
		Version: selected.Version().Unsafe(), Replicas: selected.Replicas(), Requests: totals.Requests,
		Failures: totals.Failures, Invalid: totals.Invalid, ErrorPercent: totals.ErrorPercent(), P95Millis: totals.P95Millis(),
	}
}
