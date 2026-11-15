package rollout

import "testing"

// TestTotals checks status classes, latency boundaries, and invalid records.
func TestTotals(t *testing.T) {
	var totals Totals
	for _, row := range []string{
		`{"status":200,"duration_ms":10}`,
		`{"status":404,"duration_ms":25}`,
		`{"status":503,"duration_ms":1001}`,
	} {
		if err := totals.addRecord(row); err != nil {
			t.Fatal(err)
		}
	}
	for _, row := range []string{`{`, `{"status":200}`, `{"status":99,"duration_ms":0}`} {
		if err := totals.addRecord(row); err == nil {
			t.Fatalf("accepted %s", row)
		}
	}
	if totals.Requests != 3 || totals.Failures != 1 || totals.Invalid != 3 || totals.P95Millis() != 1001 {
		t.Fatal(totals)
	}
	if totals.Buckets[0] != 1 || totals.Buckets[1] != 1 || totals.Buckets[7] != 1 {
		t.Fatal(totals.Buckets)
	}
}

// TestMergedPercentile checks the percentile rank after replica counts combine.
func TestMergedPercentile(t *testing.T) {
	one := Totals{Requests: 19, Buckets: [8]uint64{19}, MaxMillis: 10}
	two := Totals{Requests: 1, Failures: 1, Buckets: [8]uint64{0, 0, 0, 0, 0, 0, 0, 1}, MaxMillis: 2000}
	one.add(two)
	if one.P95Millis() != 10 || one.ErrorPercent() != 5 {
		t.Fatal(one)
	}
	one.add(two)
	if one.P95Millis() != 2000 {
		t.Fatal(one)
	}
}
