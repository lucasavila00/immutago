package tests

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestFrozenTransport checks persistent values and Local restrictions with both drivers.
func TestFrozenTransport(t *testing.T) {
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
		for _, target := range []string{"api", "accepted", "rejected", "implementations"} {
			t.Run(mode+"/"+target, func(t *testing.T) {
				dir := consumerModule(t, root, "frozen_transport")
				output := runBoundary(t, dir, binary, mode, target == "api" || target == "accepted", "./"+target)
				checkFixtureSnapshots(t, root, "frozen_transport/"+target, output)
			})
		}
	}
	t.Run("runtime", func(t *testing.T) {
		dir := consumerModule(t, root, "frozen_transport")
		runCommand(t, dir, true, "go", "test", "./accepted")
	})
}
