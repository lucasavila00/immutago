package tests

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestServiceEmbedding checks selected value embedding paths through both drivers.
func TestServiceEmbedding(t *testing.T) {
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
		for _, target := range []string{"valid", "pointer", "nested", "shadow", "dependency"} {
			t.Run(mode+"/"+target, func(t *testing.T) {
				dir := consumerModule(t, root, "service_embedding")
				output := runBoundary(t, dir, binary, mode, target == "valid", "./"+target)
				checkFixtureSnapshots(t, root, "service_embedding/"+target, output)
			})
		}
	}
	t.Run("runtime", func(t *testing.T) {
		dir := consumerModule(t, root, "service_embedding")
		runCommand(t, dir, true, "go", "test", "-race", "-count=1", "./valid")
	})
}
