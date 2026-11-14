package checker

import (
	"fmt"
	"go/token"
	"go/types"
	"testing"
)

// classificationQueryGraph joins recursive structs through storage fields and interface callbacks.
func classificationQueryGraph(size int, mode string) []*types.Named {
	edges := make([][]int, size)
	for i := range edges {
		edges[i] = []int{(i + 1) % size, (i*7 + 3) % size, (i*13 + 5) % size}
	}
	leaf := -1
	if mode != "ordinary" {
		leaf = size - 1
	}
	nodes := classificationGraph(edges, leaf, mode == "callback")
	for i, node := range nodes {
		method := types.NewFunc(token.NoPos, nil, "Read", classificationSignature(types.NewPointer(nodes[(i+2)%size])))
		service := types.NewInterfaceType([]*types.Func{method}, nil).Complete()
		var fields []*types.Var
		for field := range node.Underlying().(*types.Struct).Fields() {
			fields = append(fields, field)
		}
		fields = append(fields, types.NewField(token.NoPos, nil, "Service", service, false))
		node.SetUnderlying(types.NewStruct(fields, nil))
	}
	return nodes
}

// BenchmarkCoreClassification measures repeated package-local queries on connected type graphs.
func BenchmarkCoreClassification(b *testing.B) {
	for _, mode := range []string{"ordinary", "storage", "callback"} {
		for _, size := range []int{16, 64, 256} {
			b.Run(fmt.Sprintf("%s/%d", mode, size), func(b *testing.B) {
				benchmarkClassificationQueries(b, classificationQueryGraph(size, mode), mode)
			})
		}
	}
}

// benchmarkClassificationQueries includes one fresh pass cache and ten queries per type in each sample.
func benchmarkClassificationQueries(b *testing.B, nodes []*types.Named, mode string) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		var classifier Classifier
		for repeat := 0; repeat < 10; repeat++ {
			for _, node := range nodes {
				if got := classifier.Contains(node); got != (mode == "storage") {
					b.Fatal("wrong storage classification")
				}
				if got := classifier.Callable(node); got != (mode != "ordinary") {
					b.Fatal("wrong callback classification")
				}
			}
		}
	}
}

// TestClassificationInterfaceCycles compares storage and callback graphs with the original walkers.
func TestClassificationInterfaceCycles(t *testing.T) {
	for _, mode := range []string{"ordinary", "storage", "callback"} {
		nodes := classificationQueryGraph(8, mode)
		var classifier Classifier
		for _, node := range nodes {
			if got, want := classifier.Contains(node), classificationStorageOracle(node, make(map[types.Type]bool)); got != want {
				t.Fatalf("mode=%s: Contains=%t, want %t", mode, got, want)
			}
			if got, want := classifier.Callable(node), classificationCallableOracle(node, make(map[types.Type]bool)); got != want {
				t.Fatalf("mode=%s: Callable=%t, want %t", mode, got, want)
			}
		}
	}
}
