package analyzer

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand/v2"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// legacyFailure is the old copied-chain representation, used only as a reporting oracle.
type legacyFailure struct {
	location, message string
	chain             []string
}

// TestDependencyGraphMatchesLegacy compares all causes, order, and routes with the previous algorithm.
func TestDependencyGraphMatchesLegacy(t *testing.T) {
	for _, serialized := range []bool{false, true} {
		t.Run(fmt.Sprintf("serialized=%t", serialized), func(t *testing.T) {
			random := rand.New(rand.NewPCG(20, 2026))
			for range 80 {
				checkDependencyGraph(t, random, serialized)
			}
		})
	}
}

// checkDependencyGraph compares one random acyclic graph, including prefix paths and silent wrappers.
func checkDependencyGraph(t *testing.T, random *rand.Rand, serialized bool) {
	t.Helper()
	facts := make(map[string]invalidPackage)
	old := make(map[string][]legacyFailure)
	paths := []string{"example/a", "example/a/b", "example/a-a", "example/z"}
	for i := 4; i < 24; i++ {
		paths = append(paths, fmt.Sprintf("example/p%02d", i))
	}
	for i, path := range paths {
		imports := make(map[string]invalidPackage)
		oldImports := make(map[string][]legacyFailure)
		for _, imported := range paths[:i] {
			if random.IntN(4) == 0 {
				imports[imported], oldImports[imported] = facts[imported], old[imported]
			}
		}
		failures := graphLocalFailures(path, random.IntN(4))
		if len(imports) == 0 && len(failures) == 0 {
			failures = graphLocalFailures(path, 1)
		}
		old[path] = legacyFailures(path, oldImports, failures)
		fact := packageFailureFact(path, random.IntN(2) == 0, imports, failures)
		if serialized {
			fact = roundTripFact(t, fact)
		}
		facts[path] = *fact
		got, want := dependencyMessage("consumer", path, *fact), legacyMessage("consumer", path, old[path])
		if got != want {
			t.Fatalf("report changed for %s:\ngot:\n%s\nwant:\n%s", path, got, want)
		}
	}
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

// legacyFailures reproduces the old collect-sort-deduplicate algorithm without graph traversal.
func legacyFailures(path string, imports map[string][]legacyFailure, local []proofFailure) []legacyFailure {
	var failures []legacyFailure
	for _, imported := range imports {
		for _, failure := range imported {
			failure.chain = append([]string{path}, failure.chain...)
			failures = append(failures, failure)
		}
	}
	for _, failure := range local {
		failures = append(failures, legacyFailure{failure.Location, failure.Message, []string{path}})
	}
	sort.Slice(failures, func(i, j int) bool {
		a, b := failures[i], failures[j]
		if a.location != b.location {
			return a.location < b.location
		}
		if a.message != b.message {
			return a.message < b.message
		}
		return strings.Join(a.chain, " -> ") < strings.Join(b.chain, " -> ")
	})
	result := failures[:0]
	for _, failure := range failures {
		if len(result) == 0 || result[len(result)-1].location != failure.location ||
			result[len(result)-1].message != failure.message {
			result = append(result, failure)
		}
	}
	return result
}

// legacyMessage reproduces the previous public message format.
func legacyMessage(consumer, path string, failures []legacyFailure) string {
	message := "unsupported dependency " + path + ": immutability proof failed"
	for _, failure := range failures {
		chain := append([]string{consumer}, failure.chain...)
		message += "\n  import chain: " + strings.Join(chain, " -> ")
		message += "\n  original failure: " + failure.location + ": " + strings.ReplaceAll(failure.message, "\n", "\n  ")
	}
	return message + "\n  fix: correct the original dependency error above, then rerun immutago on this package"
}

// roundTripFact checks the portable representation without an in-process fact cache.
func roundTripFact(t *testing.T, fact *invalidPackage) *invalidPackage {
	t.Helper()
	data := encodeDependencyFact(t, fact)
	var decoded invalidPackage
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*fact, decoded) {
		t.Fatal("fact content changed during serialization")
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

// TestDependencyFactSharing bounds retained records and serialized data on the reported dense graph.
func TestDependencyFactSharing(t *testing.T) {
	facts := layeredDependencyFacts(20, 20, 101)
	checkSharedDependencyRecords(t, facts)
	fact := facts[len(facts)-1]
	encoded := encodeDependencyFact(t, fact)
	bound := 1024
	for _, record := range fact.Packages {
		bound += len(record.Path) + 16
		for _, path := range record.Imports {
			bound += len(path) + 8
		}
		for _, failure := range record.Failures {
			bound += len(failure.Location) + len(failure.Message) + 16
		}
	}
	if len(encoded) > bound {
		t.Fatalf("encoded graph grew beyond one copy of each record: %d > %d", len(encoded), bound)
	}
	if again := encodeDependencyFact(t, fact); !bytes.Equal(encoded, again) {
		t.Fatal("fact encoding is not deterministic")
	}
	imports := make(map[string]invalidPackage)
	for i, imported := range facts[len(facts)-20:] {
		imports[fmt.Sprintf("p19_%02d", i)] = *imported
	}
	next := encodeDependencyFact(t, packageFailureFact("next", false, imports, nil))
	for range 5 {
		if data := encodeDependencyFact(t, packageFailureFact("next", false, imports, nil)); !bytes.Equal(next, data) {
			t.Fatal("import collection changed the encoded fact order")
		}
	}
	if after := encodeDependencyFact(t, fact); !bytes.Equal(encoded, after) {
		t.Fatal("an importer changed an existing fact")
	}
	_ = roundTripFact(t, fact)
}

// checkSharedDependencyRecords requires one shared record per package across all exported facts.
func checkSharedDependencyRecords(t *testing.T, facts []*invalidPackage) {
	t.Helper()
	records := make(map[string]*failedPackage)
	for _, fact := range facts {
		seen := make(map[string]bool)
		for _, record := range fact.Packages {
			if seen[record.Path] {
				t.Fatalf("duplicate record for %s", record.Path)
			}
			seen[record.Path] = true
			if previous := records[record.Path]; previous != nil && previous != record {
				t.Fatalf("copied local failures for %s", record.Path)
			}
			records[record.Path] = record
		}
	}
	if len(records) != 400 {
		t.Fatalf("retained %d records, want 400", len(records))
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
