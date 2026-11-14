package analyzer

import (
	"fmt"
	"strings"
	"testing"
)

// TestCoreSourceStress checks large source shapes and a forbidden operation at each shape's boundary.
func TestCoreSourceStress(t *testing.T) {
	for _, shape := range []string{"records", "wide", "nested", "builders"} {
		t.Run(shape, func(t *testing.T) {
			if got := checkSource(t, stressSource(shape, 128, false)); got != "" {
				t.Fatalf("valid stress source rejected: %s", got)
			}
			want := "writes to immutable objects"
			if shape == "builders" {
				want = "use of consumed owned container"
			}
			if got := checkSource(t, stressSource(shape, 128, true)); !strings.Contains(got, want) {
				t.Fatalf("missing %q in diagnostics: %s", want, got)
			}
		})
	}
}

// stressSource constructs repeated, wide, nested, and ownership-heavy consumer programs.
func stressSource(shape string, count int, invalid bool) string {
	var source strings.Builder
	source.WriteString("type Value struct { im.Shared; N int };\n")
	switch shape {
	case "records":
		source.WriteString("func f(){\n")
		for range count {
			source.WriteString("{ value:=Value{N:1}; copy:=value; _=copy.N };\n")
		}
		if invalid {
			source.WriteString("value:=Value{};value.N=2;\n")
		}
	case "wide":
		source.WriteString("type Wide struct { im.Shared;\n")
		for index := range count {
			fmt.Fprintf(&source, "F%d int;\n", index)
		}
		source.WriteString("};func f(){ value:=Value{N:1}; wide:=Wide{\n")
		for index := range count {
			fmt.Fprintf(&source, "F%d:value.N,\n", index)
		}
		source.WriteString("};_=wide.F0;\n")
		if invalid {
			source.WriteString("wide.F0=2;\n")
		}
	case "nested":
		source.WriteString("type Node struct { im.Shared; Next *Node; N int };func f(){ value:=")
		for range count {
			source.WriteString("&Node{Next:")
		}
		source.WriteString("nil")
		source.WriteString(strings.Repeat("}", count))
		source.WriteString(";_=value.Next;\n")
		if invalid {
			source.WriteString("value.N=2;\n")
		}
	case "builders":
		source.WriteString("func f(){\n")
		for index := range count {
			source.WriteString("{ values:=[]int{1}; frozen:=im.FreezeSlice(values);_=frozen.Len();")
			if invalid && index == count-1 {
				source.WriteString("values[0]=2;")
			}
			source.WriteString("};\n")
		}
	}
	source.WriteString("}\n")
	return source.String()
}
