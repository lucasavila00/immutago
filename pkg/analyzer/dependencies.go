package analyzer

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/analysis"
)

// invalidPackage carries a failed proof across drivers that suppress dependency diagnostics.
type invalidPackage struct {
	Relevant bool
	Failures []proofFailure
}

// AFact identifies invalidPackage as a portable package-level analysis obligation.
func (*invalidPackage) AFact() {}

// proofFailure retains the original diagnostic and each import in its proof chain.
type proofFailure struct {
	Location string
	Message  string
	Chain    []string
}

// localFailure records a portable source location for a diagnostic from this package.
func localFailure(pass *analysis.Pass, diagnostic analysis.Diagnostic) proofFailure {
	position := pass.Fset.Position(diagnostic.Pos)
	message := diagnostic.Message
	for _, related := range diagnostic.Related {
		location := pass.Fset.Position(related.Pos)
		message += fmt.Sprintf("\n  related: %s/%s:%d:%d: %s", pass.Pkg.Path(),
			filepath.Base(location.Filename), location.Line, location.Column, related.Message)
	}
	return proofFailure{
		Location: fmt.Sprintf("%s/%s:%d:%d", pass.Pkg.Path(),
			filepath.Base(position.Filename), position.Line, position.Column),
		Message: message, Chain: []string{pass.Pkg.Path()},
	}
}

// invalidDependencies preserves original failures through drivers that hide dependency diagnostics.
func invalidDependencies(pass *analysis.Pass) (bool, bool, []proofFailure) {
	relevant := protectedPackage(pass)
	if pass.ImportPackageFact == nil {
		return false, relevant, nil
	}
	invalid := make(map[string]invalidPackage)
	var failures []proofFailure
	for _, imported := range pass.Pkg.Imports() {
		var fact invalidPackage
		if !pass.ImportPackageFact(imported, &fact) {
			continue
		}
		invalid[imported.Path()] = fact
		relevant = relevant || fact.Relevant
		for _, failure := range fact.Failures {
			failure.Chain = append([]string{pass.Pkg.Path()}, failure.Chain...)
			failures = append(failures, failure)
		}
	}
	if len(invalid) == 0 || !relevant {
		return len(invalid) > 0, relevant, failures
	}
	for _, file := range pass.Files {
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				continue
			}
			if fact, ok := invalid[path]; ok {
				pass.Reportf(spec.Pos(), "%s", dependencyMessage(pass.Pkg.Path(), path, fact))
			}
		}
	}
	return true, relevant, failures
}

// dependencyMessage explains each failed rule at its original source, including silent wrapper imports.
func dependencyMessage(consumer, path string, fact invalidPackage) string {
	message := "unsupported dependency " + path + ": immutability proof failed"
	for _, failure := range fact.Failures {
		chain := append([]string{consumer}, failure.Chain...)
		message += "\n  import chain: " + strings.Join(chain, " -> ")
		message += "\n  original failure: " + failure.Location + ": " + strings.ReplaceAll(failure.Message, "\n", "\n  ")
	}
	return message + "\n  fix: correct the original dependency error above, then rerun immutago on this package"
}

// protectedPackage also checks source types because vet can prune transitive package imports.
func protectedPackage(pass *analysis.Pass) bool {
	var queries checker.Classifier
	if usesImmutable(pass.Pkg) {
		return true
	}
	c := &subsetChecks{}
	for _, value := range pass.TypesInfo.Types {
		if value.Type != nil && (queries.Contains(value.Type) || c.slotRelevant(value.Type)) {
			return true
		}
	}
	return false
}

// uniqueFailures keeps one deterministic import route per original failure in diamond-shaped graphs.
func uniqueFailures(failures []proofFailure) []proofFailure {
	sort.Slice(failures, func(i, j int) bool {
		left, right := failures[i], failures[j]
		if left.Location != right.Location {
			return left.Location < right.Location
		}
		if left.Message != right.Message {
			return left.Message < right.Message
		}
		return strings.Join(left.Chain, " -> ") < strings.Join(right.Chain, " -> ")
	})
	result := failures[:0]
	for _, failure := range failures {
		if len(result) > 0 {
			previous := result[len(result)-1]
			if previous.Location == failure.Location && previous.Message == failure.Message {
				continue
			}
		}
		result = append(result, failure)
	}
	return result
}
