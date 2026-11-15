// Package rollout compares release health from live replica access logs.
package rollout

import (
	"encoding/json"
	"fmt"

	"github.com/lucasavila00/immutago/pkg/im"
)

// Totals is a copy of the counts for all complete records read so far.
type Totals struct {
	im.Copied
	Requests  uint64
	Failures  uint64
	Invalid   uint64
	Buckets   [8]uint64
	MaxMillis int
}

var latencyLimits = [7]int{10, 25, 50, 100, 250, 500, 1000}

// addRecord counts one JSON record. Invalid records block the release gate.
func (t *Totals) addRecord(line string) error {
	var record struct {
		Status int  `json:"status"`
		Millis *int `json:"duration_ms"`
	}
	err := json.Unmarshal([]byte(line), &record)
	if err == nil && (record.Status < 100 || record.Status > 599 || record.Millis == nil || *record.Millis < 0) {
		err = fmt.Errorf("record needs status 100..599 and nonnegative duration_ms")
	}
	if err != nil {
		t.Invalid++
		return err
	}
	t.Requests++
	if record.Status >= 500 {
		t.Failures++
	}
	t.MaxMillis = max(t.MaxMillis, *record.Millis)
	bucket := len(latencyLimits)
	for i, limit := range latencyLimits {
		if *record.Millis <= limit {
			bucket = i
			break
		}
	}
	t.Buckets[bucket]++
	return nil
}

// add combines disjoint replica counts.
func (t *Totals) add(other Totals) {
	t.Requests += other.Requests
	t.Failures += other.Failures
	t.Invalid += other.Invalid
	t.MaxMillis = max(t.MaxMillis, other.MaxMillis)
	for i := range t.Buckets {
		t.Buckets[i] += other.Buckets[i]
	}
}

// ErrorPercent returns the percentage of valid requests with an HTTP 5xx status.
func (t Totals) ErrorPercent() float64 {
	if t.Requests == 0 {
		return 0
	}
	return 100 * float64(t.Failures) / float64(t.Requests)
}

// P95Millis returns the upper bound of the bucket containing the 95th percentile.
// The final bucket uses the largest observed duration as its upper bound.
func (t Totals) P95Millis() int {
	if t.Requests == 0 {
		return 0
	}
	remaining := t.Requests - t.Requests/20
	for i, count := range t.Buckets {
		if count >= remaining {
			if i == len(latencyLimits) {
				return t.MaxMillis
			}
			return latencyLimits[i]
		}
		remaining -= count
	}
	return 0
}
