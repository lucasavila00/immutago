package tests

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestServiceErrors checks ordinary errors and protected result pairs through both drivers.
func TestServiceErrors(t *testing.T) {
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
		for _, target := range []string{"api", "valid", "ordinary", "signatures", "shadow", "dependency", "uses"} {
			t.Run(mode+"/"+target, func(t *testing.T) {
				dir := consumerModule(t, root, "service_errors")
				accepted := target == "api" || target == "valid" || target == "ordinary"
				output := runBoundary(t, dir, binary, mode, accepted, "./"+target)
				checkFixtureSnapshots(t, root, "service_errors/"+target, output)
			})
		}
	}
	t.Run("runtime", func(t *testing.T) {
		dir := consumerModule(t, root, "service_errors")
		runCommand(t, dir, true, "go", "test", "-race", "-count=1", "./valid")
	})
}
