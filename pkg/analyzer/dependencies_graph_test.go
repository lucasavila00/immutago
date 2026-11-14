package analyzer

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
	"testing"
)

// failureNode retains all original errors for an independent graph traversal.
type failureNode struct {
	imports  []string
	failures []proofFailure
}

// TestDependencyWitnessMatchesGraph compares one propagated cause with every reachable original error.
func TestDependencyWitnessMatchesGraph(t *testing.T) {
	for _, serialized := range []bool{false, true} {
		t.Run(fmt.Sprintf("serialized=%t", serialized), func(t *testing.T) {
			random := rand.New(rand.NewPCG(28, 2026))
			for range 80 {
				checkDependencyGraph(t, random, serialized)
			}
		})
	}
}

// checkDependencyGraph checks an acyclic graph with prefix paths, shared causes, and silent wrappers.
func checkDependencyGraph(t *testing.T, random *rand.Rand, serialized bool) {
	t.Helper()
	facts := make(map[string]invalidPackage)
	graph := make(map[string]failureNode)
	paths := []string{"example/a", "example/a/b", "example/a-a", "example/z"}
	for i := 4; i < 24; i++ {
		paths = append(paths, fmt.Sprintf("example/p%02d", i))
	}
	for i, path := range paths {
		imports := make(map[string]invalidPackage)
		node := failureNode{failures: graphLocalFailures(path, random.IntN(4))}
		for _, imported := range paths[:i] {
			if random.IntN(4) == 0 {
				imports[imported] = facts[imported]
				node.imports = append(node.imports, imported)
			}
		}
		if len(imports) == 0 && len(node.failures) == 0 {
			node.failures = graphLocalFailures(path, 1)
		}
		graph[path] = node
		relevant := random.IntN(2) == 0
		fact := packageFailureFact(path, relevant, imports, node.failures)
		if serialized {
			fact = roundTripFact(t, fact)
		}
		facts[path] = *fact
		want := firstGraphFailure(graph, path)
		want.Relevant = relevant
		if *fact != want {
			t.Fatalf("witness changed for %s: got %+v, want %+v", path, *fact, want)
		}
	}
}

// firstGraphFailure sorts all reachable errors instead of reducing imported witnesses.
func firstGraphFailure(graph map[string]failureNode, path string) invalidPackage {
	var failures []invalidPackage
	seen := make(map[string]bool)
	pending := []string{path}
	for len(pending) > 0 {
		path = pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if seen[path] {
			continue
		}
		seen[path] = true
		node := graph[path]
		pending = append(pending, node.imports...)
		for _, failure := range node.failures {
			failures = append(failures, invalidPackage{Origin: path, Failure: failure})
		}
	}
	sort.Slice(failures, func(i, j int) bool {
		a, b := failures[i], failures[j]
		if a.Failure.Location != b.Failure.Location {
			return a.Failure.Location < b.Failure.Location
		}
		if a.Failure.Message != b.Failure.Message {
			return a.Failure.Message < b.Failure.Message
		}
		return a.Origin < b.Origin
	})
	return failures[0]
}

// graphLocalFailures includes repeated errors, different errors at one location, and multiline text.
func graphLocalFailures(path string, count int) []proofFailure {
	var failures []proofFailure
	for i := range count {
		failure := proofFailure{Location: fmt.Sprintf("%s/%c.go:1:1", path, 'z'-i),
			Message: fmt.Sprintf("rule %d\n  related: original detail", i%2)}
		failures = append(failures, failure, failure,
			proofFailure{Location: failure.Location, Message: "another rule"})
	}
	return failures
}

// roundTripFact checks the complete portable value without an in-process fact cache.
func roundTripFact(t *testing.T, fact *invalidPackage) *invalidPackage {
	t.Helper()
	data := encodeDependencyFact(t, fact)
	var decoded invalidPackage
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if *fact != decoded {
		t.Fatalf("fact changed during serialization: got %+v, want %+v", decoded, *fact)
	}
	return &decoded
}

// encodeDependencyFact encodes one value for fact round trips and size checks.
func encodeDependencyFact(t testing.TB, value any) []byte {
	t.Helper()
	var data bytes.Buffer
	if err := gob.NewEncoder(&data).Encode(value); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

// TestDependencyFactSize requires constant fact bytes across chains, wide layers, and dense graphs.
func TestDependencyFactSize(t *testing.T) {
	for _, shape := range [][2]int{{512, 1}, {1, 64}, {4, 4}, {8, 8}, {12, 12}, {16, 16}, {20, 20}, {24, 24}} {
		t.Run(fmt.Sprintf("layers=%d/width=%d", shape[0], shape[1]), func(t *testing.T) {
			facts := layeredDependencyFacts(shape[0], shape[1], 101)
			// These witnesses have equal text lengths. Depth, edges, and omitted causes add no bytes.
			one := len(encodeDependencyFact(t, facts[0]))
			total := 0
			for _, fact := range facts {
				data := encodeDependencyFact(t, fact)
				if len(data) != one {
					t.Fatalf("fact size grew with the graph: %d bytes, want %d", len(data), one)
				}
				total += len(data)
			}
			t.Logf("packages=%d total=%d largest=%d bytes", len(facts), total, one)
			_ = roundTripFact(t, facts[len(facts)-1])
		})
	}
}

// TestDependencyWitnessIsExact keeps long original text and its origin through silent wrappers.
func TestDependencyWitnessIsExact(t *testing.T) {
	original := &invalidPackage{Origin: "example/origin", Failure: proofFailure{
		Location: "example/origin/" + strings.Repeat("source", 500) + ".go:12:3",
		Message:  strings.Repeat("original detail\n  related: more detail\n", 500),
	}}
	fact := original
	for i := range 100 {
		fact = roundTripFact(t, packageFailureFact(fmt.Sprintf("example/wrapper%d", i), false,
			map[string]invalidPackage{"previous": *fact}, nil))
	}
	if *fact != *original {
		t.Fatal("silent wrappers changed or shortened the original cause")
	}
	message := dependencyMessage("example/wrapper99", *fact)
	for _, want := range []string{"representative failure: " + original.Failure.Location,
		strings.ReplaceAll(original.Failure.Message, "\n", "\n  "),
		"other dependency errors may exist", "run immutago example/origin,"} {
		if !strings.Contains(message, want) {
			t.Fatalf("missing original detail or command: %q", want)
		}
	}
}

// layeredDependencyFacts builds the issue's dense import layers without parsing or type checking.
func layeredDependencyFacts(layers, width, failures int) []*invalidPackage {
	var facts []*invalidPackage
	previous := make(map[string]invalidPackage)
	for layer := range layers {
		current := make(map[string]invalidPackage)
		for column := range width {
			path := fmt.Sprintf("p%02d_%02d", layer, column)
			local := make([]proofFailure, failures)
			for i := range local {
				local[i] = proofFailure{Location: fmt.Sprintf("%s/p.go:%d:1", path, i+1),
					Message: "unsupported unsafe pointer; immutability cannot be proved\n  fix: use typed values"}
			}
			fact := packageFailureFact(path, false, previous, local)
			facts = append(facts, fact)
			current[path] = *fact
		}
		previous = current
	}
	return facts
}
