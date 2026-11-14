package analyzer

import (
	"cmp"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/analysis"
)

// invalidPackage carries one original failure across drivers that hide dependency diagnostics.
type invalidPackage struct {
	Relevant bool
	Origin   string
	Failure  proofFailure
}

// AFact identifies invalidPackage as a portable package-level analysis obligation.
func (*invalidPackage) AFact() {}

// proofFailure retains an exact original diagnostic and its portable source location.
type proofFailure struct {
	Location string
	Message  string
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
		Message: message,
	}
}

// invalidDependencies reports one original cause for each failed direct import when relevant.
func invalidDependencies(pass *analysis.Pass) (bool, bool, map[string]invalidPackage) {
	relevant, invalid := collectDependencyFacts(pass)
	if len(invalid) == 0 || !relevant {
		return len(invalid) > 0, relevant, invalid
	}
	for _, file := range pass.Files {
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				continue
			}
			if fact, ok := invalid[path]; ok {
				pass.Reportf(spec.Pos(), "%s", dependencyMessage(path, fact))
			}
		}
	}
	return true, relevant, invalid
}

// collectDependencyFacts reads direct failures without emitting dependency diagnostics.
func collectDependencyFacts(pass *analysis.Pass) (bool, map[string]invalidPackage) {
	relevant := protectedPackage(pass)
	if pass.ImportPackageFact == nil {
		return relevant, nil
	}
	invalid := make(map[string]invalidPackage)
	for _, imported := range pass.Pkg.Imports() {
		var fact invalidPackage
		if !pass.ImportPackageFact(imported, &fact) {
			continue
		}
		invalid[imported.Path()] = fact
		relevant = relevant || fact.Relevant
	}
	return relevant, invalid
}

// packageFailureFact selects the first original failure without copying the failed import graph.
// Taking the minimum of local and imported causes preserves the same choice across silent wrappers.
func packageFailureFact(path string, relevant bool, imports map[string]invalidPackage,
	failures []proofFailure) *invalidPackage {
	fact := &invalidPackage{Relevant: relevant}
	consider := func(origin string, failure proofFailure) {
		order := compareFailures(failure, fact.Failure)
		if fact.Origin == "" || order < 0 || order == 0 && origin < fact.Origin {
			fact.Origin, fact.Failure = origin, failure
		}
	}
	for _, failure := range failures {
		consider(path, failure)
	}
	for _, imported := range imports {
		consider(imported.Origin, imported.Failure)
	}
	return fact
}

// dependencyMessage identifies the failed import and one cause, with a command for the original package.
func dependencyMessage(path string, fact invalidPackage) string {
	var message strings.Builder
	message.WriteString("unsupported dependency " + path + ": immutability proof failed")
	message.WriteString("\n  representative failure: " + fact.Failure.Location + ": ")
	message.WriteString(strings.ReplaceAll(fact.Failure.Message, "\n", "\n  "))
	message.WriteString("\n  note: other dependency errors may exist")
	message.WriteString("\n  fix: run immutago " + fact.Origin +
		", correct its errors, then rerun immutago on this package")
	return message.String()
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

// compareFailures preserves diagnostic order by portable source location and message.
func compareFailures(a, b proofFailure) int {
	if order := cmp.Compare(a.Location, b.Location); order != 0 {
		return order
	}
	return cmp.Compare(a.Message, b.Message)
}
