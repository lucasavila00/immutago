package tests

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestServiceCopyContract checks the standard vet signal without adding an immutago copy rule.
func TestServiceCopyContract(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	dir := consumerModule(t, root, "service_copy")
	for _, fixture := range []string{"accepted", "rejected"} {
		t.Run(fixture, func(t *testing.T) {
			output := runCommand(t, dir, fixture == "accepted", "go", "vet", "./"+fixture)
			checkFixtureSnapshots(t, root, "service_copy/"+fixture, output)
		})
	}
	binary := filepath.Join(t.TempDir(), "immutago")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	runCommand(t, root, true, "go", "build", "-o", binary, "./cmd/immutago")
	for _, mode := range []string{"cli", "vet"} {
		t.Run(mode, func(t *testing.T) {
			if output := runBoundary(t, dir, binary, mode, true, "./..."); output != "" {
				t.Fatalf("immutago checked an ordinary Go copy contract: %s", output)
			}
		})
	}
}
