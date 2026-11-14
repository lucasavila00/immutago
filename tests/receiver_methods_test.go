package tests

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestReceiverMethods checks exact receiver calls and imported method bodies through both drivers.
func TestReceiverMethods(t *testing.T) {
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
			"accepted", "rejected", "api", "consumer", "rejectedconsumer", "invalid", "invalidconsumer",
		} {
			t.Run(mode+"/"+target, func(t *testing.T) {
				dir := consumerModule(t, root, "receiver_methods")
				valid := target == "accepted" || target == "api" || target == "consumer"
				output := runBoundary(t, dir, binary, mode, valid, "./"+target)
				checkFixtureSnapshots(t, root, "receiver_methods/"+target, output)
			})
		}
	}
}
