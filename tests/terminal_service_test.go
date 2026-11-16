package tests

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestTerminalServiceRules checks the shared im rules and imported terminal services.
func TestTerminalServiceRules(t *testing.T) {
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
		for _, target := range []string{"api", "valid", "results", "storage", "calls", "dependency"} {
			t.Run(mode+"/"+target, func(t *testing.T) {
				dir := consumerModule(t, root, "terminal_service")
				valid := target == "api" || target == "valid"
				output := runBoundary(t, dir, binary, mode, valid, "./"+target)
				checkFixtureSnapshots(t, root, "terminal_service/"+target, output)
			})
		}
	}
}
