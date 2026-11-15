package tests

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestFrozenInterfaces checks interface contracts, projections, and dependency failures with both drivers.
func TestFrozenInterfaces(t *testing.T) {
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
			"api", "implementation", "accepted", "rejected", "invalid", "invalidconsumer", "contracts",
			"privateapi", "privateconsumer", "result_rejected",
		} {
			t.Run(mode+"/"+target, func(t *testing.T) {
				dir := consumerModule(t, root, "frozen_interfaces")
				valid := target == "api" || target == "implementation" ||
					target == "accepted" || target == "privateapi"
				output := runBoundary(t, dir, binary, mode, valid, "./"+target)
				checkFixtureSnapshots(t, root, "frozen_interfaces/"+target, output)
			})
		}
	}
	t.Run("compiler_seal", func(t *testing.T) {
		dir := consumerModule(t, root, "frozen_interfaces")
		writeFile(t, filepath.Join(dir, "rejected", "plain.go"), `package rejected
import "example.com/consumer/api"
type Plain struct{}
func (Plain) Score([]float32) float64 { return 0 }
var _ api.Scorer = Plain{}
`)
		output := runCommand(t, dir, false, binary, "./rejected")
		path := filepath.Join(root, "tests/testdata/frozen_interfaces/compiler_seal/output.txt")
		checkSnapshot(t, path, normalizeOutput(output))
	})
	t.Run("runtime", func(t *testing.T) {
		dir := consumerModule(t, root, "frozen_interfaces")
		runCommand(t, dir, true, "go", "test", "./accepted")
	})
}
