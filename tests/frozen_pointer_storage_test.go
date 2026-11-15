package tests

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestFrozenPointerStorage checks pointer carriers and target writes through both drivers.
func TestFrozenPointerStorage(t *testing.T) {
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
		for _, target := range []string{"api", "valid", "invalid", "dependency", "embedded", "embedded_dependency"} {
			t.Run(mode+"/"+target, func(t *testing.T) {
				dir := consumerModule(t, root, "frozen_pointer_storage")
				output := runBoundary(t, dir, binary, mode, target == "api" || target == "valid", "./"+target)
				checkFixtureSnapshots(t, root, "frozen_pointer_storage/"+target, output)
			})
		}
	}
	t.Run("runtime", func(t *testing.T) {
		dir := consumerModule(t, root, "frozen_pointer_storage")
		runCommand(t, dir, true, "go", "test", "-race", "-count=1", "./valid")
	})
}
