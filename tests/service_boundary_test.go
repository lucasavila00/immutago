package tests

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestServiceBoundary checks service-only packages and imported Ref contracts through both drivers.
func TestServiceBoundary(t *testing.T) {
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
		for _, target := range []string{
			"api", "valid", "wrapper", "sealedapi", "onlywrapper", "mixedallowed",
			"direct", "capture", "contract", "contracts",
			"dependency", "invalidresults",
		} {
			t.Run(mode+"/"+target, func(t *testing.T) {
				dir := consumerModule(t, root, "service_boundary")
				accepted := target == "api" || target == "valid" || target == "wrapper" || target == "mixedallowed" ||
					target == "sealedapi" || target == "onlywrapper" || target == "contract" || target == "dependency"
				output := runBoundary(t, dir, binary, mode, accepted, "./"+target)
				checkFixtureSnapshots(t, root, "service_boundary/"+target, output)
			})
		}
	}
	t.Run("runtime", func(t *testing.T) {
		dir := consumerModule(t, root, "service_boundary")
		runCommand(t, dir, true, "go", "test", "-race", "-count=1", "./valid")
	})
}
