package analyzer

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
)

// slotGraphPass uses value-linked cycles in ordinary or checked packages.
func slotGraphPass(t testing.TB, count int, checked bool) *analysis.Pass {
	t.Helper()
	if !checked {
		return ordinaryGraphPass(t, count)
	}
	declarations := strings.TrimPrefix(ordinaryTypeGraph(count), "package ordinary\n")
	pass := sourcePass(t, "type Checked struct { im.Shared }\n"+declarations)
	pass.Report = func(d analysis.Diagnostic) { t.Fatalf("valid graph rejected: %s", d.Message) }
	return pass
}

// TestSlotTypeGraph keeps ordinary value-linked cycles valid inside checked packages.
func TestSlotTypeGraph(t *testing.T) {
	for _, checked := range []bool{false, true} {
		if _, err := run(slotGraphPass(t, 8, checked)); err != nil {
			t.Fatal(err)
		}
	}
}

// BenchmarkCoreSlotGraph measures fresh wrapper queries over shared value-linked types.
func BenchmarkCoreSlotGraph(b *testing.B) {
	for _, checked := range []bool{false, true} {
		kind := "ordinary"
		if checked {
			kind = "checked"
		}
		for _, count := range []int{512, 1024, 2048} {
			b.Run(fmt.Sprintf("%s/%d", kind, count), func(b *testing.B) {
				pass := slotGraphPass(b, count, checked)
				b.ReportAllocs()
				for b.Loop() {
					if _, err := run(pass); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
