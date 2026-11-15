package analyzer

import (
	"strings"
	"testing"
)

// TestFrozenNamedFields rejects compiler-promoted data methods before transport.
func TestFrozenNamedFields(t *testing.T) {
	const prefix = `type Marker=im.Frozen;type Count int;func(c *Count)Inc(){*c++};
type Stats struct{im.Copied;N int};func(s *Stats)Inc(){s.N++};
type Child struct{im.Frozen;N int};type Alias=Count;
func bump[T interface{Inc()}](value T){value.Inc()};`
	for _, field := range []string{"Count", "Alias", "Stats", "*Count", "Child", "*Child"} {
		source := prefix + `type Value struct{Marker;` + field + `}`
		if got := checkSource(t, source); !strings.Contains(got, "Frozen data fields must be named") {
			t.Fatalf("embedded %s accepted: %s", field, got)
		}
	}
	for _, body := range []string{
		`var value interface{Inc()}=p;value.Inc()`,
		`bump(p)`,
		`callback(p)`,
	} {
		source := prefix + `type Value struct{Marker;Count};
func f(p *Value,callback func(interface{Inc()})){` + body + `}`
		if got := checkSource(t, source); !strings.Contains(got, "Frozen data fields must be named") {
			t.Fatalf("embedded method escaped through %s: %s", body, got)
		}
	}
}

// TestFrozenNamedFieldReplacement keeps ordinary copies and other marker rules.
func TestFrozenNamedFieldReplacement(t *testing.T) {
	source := `type Marker=im.Frozen;type Count int;func(c *Count)Inc(){*c++};
type Stats struct{im.Copied;N int};
type Value struct{Marker;Count Count;Stats Stats};
type Local struct{im.Local;Stats};type Copy struct{im.Copied;Stats};
func f(p *Value){count:=p.Count;var value interface{Inc()}=&count;value.Inc()}`
	if got := checkSource(t, source); got != "" {
		t.Fatalf("named replacement or other marker rejected: %s", got)
	}
}
