package tests

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestUncheckedValueBoundary checks the declaring-package rule through the CLI and vet.
func TestUncheckedValueBoundary(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "immutago")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	runCommand(t, root, true, "go", "build", "-o", binary, "./cmd/immutago")
	for _, mode := range []string{"cli", "vet"} {
		for _, target := range []string{"api", "valid", "foreign"} {
			t.Run(mode+"/"+target, func(t *testing.T) {
				dir := consumerModule(t, root, "unchecked_value_boundary")
				output := runBoundary(t, dir, binary, mode, target != "foreign", "./"+target)
				checkFixtureSnapshots(t, root, "unchecked_value_boundary/"+target, output)
			})
		}
	}
}
