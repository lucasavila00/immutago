//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package rollout

import (
	"fmt"

	"github.com/lucasavila00/immutago/pkg/im"
)

// Release keeps the release name and all its counts in one immutable value.
type Release struct {
	im.Frozen
	Version  string
	Replicas int
	Totals   Totals
}

// Report is one complete selection and decision. A request keeps its chosen version.
type Report struct {
	im.Frozen
	Primary       Release
	Candidate     *Release
	Verdict       string
	Reason        string
	Problem       string
	PolicyVersion string
	PolicyProblem string
}

// evaluate compares counts after both releases have enough valid requests.
func evaluate(primary, candidate Totals, rules Limits) (string, string) {
	if primary.Invalid != 0 || candidate.Invalid != 0 {
		return "invalid_logs", "Select corrected log files before evaluating this release."
	}
	if primary.Requests < rules.MinRequests || candidate.Requests < rules.MinRequests {
		return "waiting", fmt.Sprintf("Each release needs at least %d requests.", rules.MinRequests)
	}
	if candidate.ErrorPercent()-primary.ErrorPercent() > rules.MaxErrorIncreasePercent {
		return "fail", "The candidate's error increase exceeds the limit."
	}
	if candidate.P95Millis()-primary.P95Millis() > rules.MaxP95IncreaseMillis {
		return "fail", "The candidate's p95 latency increase exceeds the limit."
	}
	return "pass", "The candidate is within both limits."
}
