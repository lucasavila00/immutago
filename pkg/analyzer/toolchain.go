package analyzer

import (
	"go/types"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/analysis"
)

// standardLibrary identifies compiler-supplied source by its SDK directory.
// Compiler packages are outside the local marked-storage checking boundary.
func standardLibrary(pass *analysis.Pass) bool {
	sdk, err := compilerSourceDirectory()
	if err != nil {
		return false
	}
	for _, file := range pass.Files {
		path := pass.Fset.PositionFor(file.Pos(), false).Filename
		relative, err := filepath.Rel(sdk, path)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) &&
			!filepath.IsAbs(relative) {
			return true
		}
	}
	return false
}

// compilerSourceDirectory uses the active Go toolchain rather than the analyzer's build-machine path.
var compilerSourceDirectory = sync.OnceValues(func() (string, error) {
	output, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return "", err
	}
	return filepath.Join(strings.TrimSpace(string(output)), "src"), nil
})

// testMainPackage identifies the Go test runner by its compiler-checked internal import.
func testMainPackage(pkg *types.Package) bool {
	if pkg.Name() != "main" || !strings.HasSuffix(pkg.Path(), ".test") {
		return false
	}
	for _, imported := range pkg.Imports() {
		if imported.Path() == "testing/internal/testdeps" {
			return true
		}
	}
	return false
}

// usesImmutable reports a direct import of the fixed immutable value or process package.
func usesImmutable(pkg *types.Package) bool {
	for _, imported := range pkg.Imports() {
		if imported.Path() == checker.LibraryPath || imported.Path() == checker.RuntimePath {
			return true
		}
	}
	return false
}
