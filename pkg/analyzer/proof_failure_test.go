package analyzer

import (
	"strings"
	"testing"
)

// TestOperationFailureRepair preserves invalid type repairs and replaces valid-payload operation repairs.
func TestOperationFailureRepair(t *testing.T) {
	cases := []struct{ name, source, repair string }{
		{"interface marker", `func f(v im.FrozenInterface){p:=im.ProjectionOf(v);_=im.Value(p)}`,
			"embed im.FrozenInterface in a named interface; use that interface as the payload"},
		{"marker alias", `type Marker=im.FrozenInterface;
func f(v Marker){p:=im.ProjectionOf(v);_=im.Value(p)}`,
			"embed im.FrozenInterface in a named interface; use that interface as the payload"},
		{"Clone string", `func f(p im.SliceProjection[string]){_=p.Clone()}`,
			"iterate the projection with All; read or clone each element into a new native collection"},
		{"Clone string alias", `type Text=string;func f(p im.SliceProjection[Text]){_=p.Clone()}`,
			"iterate the projection with All; read or clone each element into a new native collection"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := checkSource(t, tc.source)
			operations := strings.Split(got, "unsupported immutable operation: ")[1:]
			if len(operations) == 0 {
				t.Fatalf("missing operation diagnostic: %s", got)
			}
			for _, operation := range operations {
				if !strings.Contains(operation, "\n  fix: "+tc.repair) {
					t.Fatalf("repair lost in operation: %s", operation)
				}
			}
		})
	}
}
