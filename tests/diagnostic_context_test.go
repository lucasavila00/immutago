package tests

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestDiagnosticContext checks rejected operations and the exact proposed repairs through both drivers.
func TestDiagnosticContext(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "immutago")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	runCommand(t, root, true, "go", "build", "-o", binary, "./cmd/immutago")
	for _, fixture := range []string{"accepted", "rejected"} {
		dir := consumerModule(t, root, fixture)
		for _, mode := range []string{"cli", "vet"} {
			t.Run(fixture+"/"+mode, func(t *testing.T) {
				output := runBoundary(t, dir, binary, mode, fixture == "accepted", "./diagnostic_context")
				checkFixtureSnapshots(t, root, fixture+"/diagnostic_context", output)
			})
		}
	}
}
