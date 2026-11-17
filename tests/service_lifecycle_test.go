package tests

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestServiceLifecycle checks declared lifecycle receivers through both drivers.
func TestServiceLifecycle(t *testing.T) {
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
		for _, target := range []string{"hooks", "valid", "rejected", "protected", "badapi", "dependency"} {
			t.Run(mode+"/"+target, func(t *testing.T) {
				dir := consumerModule(t, root, "service_lifecycle")
				accepted := target == "hooks" || target == "valid"
				output := runBoundary(t, dir, binary, mode, accepted, "./"+target)
				checkFixtureSnapshots(t, root, "service_lifecycle/"+target, output)
			})
		}
	}
	t.Run("runtime", func(t *testing.T) {
		dir := consumerModule(t, root, "service_lifecycle")
		runCommand(t, dir, true, "go", "test", "-race", "-count=1", "./valid")
	})
}
