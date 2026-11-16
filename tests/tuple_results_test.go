package tests

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestTupleResults checks component rules through CLI, vet, imports, and runtime calls.
func TestTupleResults(t *testing.T) {
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
		for _, target := range []string{"api", "plain", "accepted", "badapi", "rejected", "dependency"} {
			t.Run(mode+"/"+target, func(t *testing.T) {
				dir := consumerModule(t, root, "tuple_results")
				accepted := target == "api" || target == "plain" || target == "accepted"
				output := runBoundary(t, dir, binary, mode, accepted, "./"+target)
				checkFixtureSnapshots(t, root, "tuple_results/"+target, output)
			})
		}
	}
	t.Run("runtime", func(t *testing.T) {
		dir := consumerModule(t, root, "tuple_results")
		runCommand(t, dir, true, "go", "test", "-race", "-count=1", "./accepted")
	})
}
