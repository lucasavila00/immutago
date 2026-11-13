package borrow

import (
	"golang.org/x/tools/go/analysis"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// IsStandardLibrary identifies compiler-supplied source by its SDK directory.
// Compiler packages are outside the local marked-storage checking boundary.
func IsStandardLibrary(pass *analysis.Pass) bool {
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
