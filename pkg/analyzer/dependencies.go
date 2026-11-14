package analyzer

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/analysis"
)

// invalidPackage carries a failed proof across drivers that suppress dependency diagnostics.
type invalidPackage struct {
	Relevant bool
	Packages []*failedPackage
}

// AFact identifies invalidPackage as a portable package-level analysis obligation.
func (*invalidPackage) AFact() {}

// failedPackage stores one package's local failures and direct invalid imports.
// Records are shared between facts and are not changed after export.
type failedPackage struct {
	Path     string
	Imports  []string
	Failures []proofFailure
}

// proofFailure retains an original diagnostic without a copied import chain.
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

// invalidDependencies preserves original failures through drivers that hide dependency diagnostics.
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
				pass.Reportf(spec.Pos(), "%s", dependencyMessage(pass.Pkg.Path(), path, fact))
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

// packageFailureFact joins flat records before export, without copying local failures or import chains.
func packageFailureFact(path string, relevant bool, imports map[string]invalidPackage,
	failures []proofFailure) *invalidPackage {
	current := &failedPackage{Path: path, Failures: uniqueFailures(failures)}
	records := make(map[string]*failedPackage)
	for imported, fact := range imports {
		current.Imports = append(current.Imports, imported)
		for _, record := range fact.Packages {
			records[record.Path] = record
		}
	}
	slices.Sort(current.Imports)
	records[path] = current
	fact := &invalidPackage{Relevant: relevant, Packages: make([]*failedPackage, 0, len(records))}
	for _, record := range records {
		fact.Packages = append(fact.Packages, record)
	}
	slices.SortFunc(fact.Packages, func(a, b *failedPackage) int { return cmp.Compare(a.Path, b.Path) })
	return fact
}

// routedFailure connects a reported local error to its origin in the import graph.
type routedFailure struct {
	failure proofFailure
	origin  string
}

// dependencyMessage builds import chains only when a relevant consumer reports a failure.
func dependencyMessage(consumer, path string, fact invalidPackage) string {
	parents := dependencyParents(path, fact)
	var failures []routedFailure
	for _, record := range fact.Packages {
		for _, failure := range record.Failures {
			failures = append(failures, routedFailure{failure: failure, origin: record.Path})
		}
	}
	slices.SortFunc(failures, func(a, b routedFailure) int { return compareFailures(a.failure, b.failure) })
	chains := make(map[string]string)
	var message strings.Builder
	message.WriteString("unsupported dependency " + path + ": immutability proof failed")
	for _, failure := range failures {
		chain, ok := chains[failure.origin]
		if !ok {
			chain = dependencyChain(consumer, failure.origin, parents)
			chains[failure.origin] = chain
		}
		message.WriteString("\n  import chain: " + chain)
		message.WriteString("\n  original failure: " + failure.failure.Location + ": ")
		message.WriteString(strings.ReplaceAll(failure.failure.Message, "\n", "\n  "))
	}
	message.WriteString("\n  fix: correct the original dependency error above, then rerun immutago on this package")
	return message.String()
}

// dependencyParents selects the first lexical route through the acyclic import graph.
func dependencyParents(path string, fact invalidPackage) map[string]string {
	records := make(map[string]*failedPackage, len(fact.Packages))
	for _, record := range fact.Packages {
		records[record.Path] = record
	}
	type edge struct{ path, parent string }
	pending := []edge{{path: path}}
	parents := make(map[string]string, len(records))
	for len(pending) > 0 {
		last := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if _, seen := parents[last.path]; seen {
			continue
		}
		parents[last.path] = last.parent
		imports := records[last.path].Imports
		for i := len(imports) - 1; i >= 0; i-- {
			pending = append(pending, edge{path: imports[i], parent: last.path})
		}
	}
	return parents
}

// dependencyChain renders one route from its parent links, including the reporting consumer.
func dependencyChain(consumer, origin string, parents map[string]string) string {
	var reversed []string
	for path := origin; path != ""; path = parents[path] {
		reversed = append(reversed, path)
	}
	var chain strings.Builder
	chain.WriteString(consumer)
	for i := len(reversed) - 1; i >= 0; i-- {
		chain.WriteString(" -> " + reversed[i])
	}
	return chain.String()
}

// protectedPackage also checks source types because vet can prune transitive package imports.
func protectedPackage(pass *analysis.Pass) bool {
	var queries checker.Classifier
	if usesImmutable(pass.Pkg) {
		return true
	}
	for _, value := range pass.TypesInfo.Types {
		if value.Type != nil && queries.Contains(value.Type) {
			return true
		}
	}
	return false
}

// uniqueFailures sorts and removes duplicate local diagnostics before their first export.
func uniqueFailures(failures []proofFailure) []proofFailure {
	slices.SortFunc(failures, compareFailures)
	return slices.Compact(failures)
}

// compareFailures preserves diagnostic order by portable source location and message.
func compareFailures(a, b proofFailure) int {
	if order := cmp.Compare(a.Location, b.Location); order != 0 {
		return order
	}
	return cmp.Compare(a.Message, b.Message)
}
