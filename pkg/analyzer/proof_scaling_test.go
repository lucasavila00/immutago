package analyzer

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
)

// markedTypeGraph reproduces the linked marked types and ten operations per function from issue 27.
func markedTypeGraph(count int, invalid bool) string {
	var source strings.Builder
	for i := range count {
		fmt.Fprintf(&source, "type M%d struct { im.Shared; Next *M%d; Many im.FrozenSlice[*M%d]; "+
			"ByKey im.FrozenMap[string,*M%d]; Name string; N int;", i, (i+1)%count, (i*7+3)%count, (i*13+5)%count)
		if invalid && i == 0 {
			source.WriteString(" Bad []int;")
		}
		source.WriteString("}\n")
	}
	for i := range count {
		writeMarkedFunction(&source, count, i)
	}
	return source.String()
}

// writeMarkedFunction emits the issue's construction, freeze, and read operations without compiler setup.
func writeMarkedFunction(source *strings.Builder, count, i int) {
	next, many, key := (i+1)%count, (i*7+3)%count, (i*13+5)%count
	fmt.Fprintf(source, "func F%d(next *M%d,a,c *M%d,k *M%d)(*M%d,int){total:=0;\n", i, next, many, key, i)
	for j := range 10 {
		fmt.Fprintf(source, "many%d:=[]*M%d{a};many%d=append(many%d,c);keys%d:=map[string]*M%d{\"k\":k};\n",
			j, many, j, j, j, key)
		fmt.Fprintf(source, "fm%d:=im.FreezeSlice(many%d);fk%d:=im.FreezeMap(keys%d);\n", j, j, j, j)
		fmt.Fprintf(source, "v%d:=&M%d{Next:next,Many:fm%d,ByKey:fk%d,Name:\"x\",N:%d};\n", j, i, j, j, j)
		fmt.Fprintf(source, "if got,ok:=v%d.ByKey.Get(\"k\");ok{total+=got.N+v%d.Many.Len()+v%d.Next.N};\n", j, j, j)
	}
	fmt.Fprintf(source, "return &M%d{Next:next,Name:\"r\",N:total},total}\n", i)
}

// TestMarkedTypeGraph checks both accepted linked objects and a reachable invalid field.
func TestMarkedTypeGraph(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		pass := sourcePass(t, markedTypeGraph(4, invalid))
		reports := 0
		pass.Report = func(analysis.Diagnostic) { reports++ }
		if _, err := run(pass); err != nil {
			t.Fatal(err)
		}
		if (reports != 0) != invalid {
			t.Fatalf("invalid=%t: reports=%d", invalid, reports)
		}
	}
}

// BenchmarkCoreMarked measures the issue's accepted workload through the complete analyzer pass.
func BenchmarkCoreMarked(b *testing.B) {
	for _, count := range []int{8, 16, 32, 64, 128, 256} {
		b.Run(fmt.Sprint(count), func(b *testing.B) { benchmarkMarkedGraph(b, count, false) })
	}
}

// BenchmarkCoreMarkedInvalid keeps the cost of distinct invalid roots and exact diagnostics visible.
func BenchmarkCoreMarkedInvalid(b *testing.B) {
	for _, count := range []int{8, 16, 32, 64, 128} {
		b.Run(fmt.Sprint(count), func(b *testing.B) { benchmarkMarkedGraph(b, count, true) })
	}
}

// benchmarkMarkedGraph excludes parsing and type checking from each measured analyzer pass.
func benchmarkMarkedGraph(b *testing.B, count int, invalid bool) {
	b.Helper()
	pass := sourcePass(b, markedTypeGraph(count, invalid))
	pass.Report = func(d analysis.Diagnostic) {
		if !invalid {
			b.Fatalf("accepted source rejected: %s", d.Message)
		}
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := run(pass); err != nil {
			b.Fatal(err)
		}
	}
}
