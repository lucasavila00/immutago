package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestBuildTags checks the documented build-tag commands with both drivers.
func TestBuildTags(t *testing.T) {
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
		t.Run(mode+"/default", func(t *testing.T) {
			dir := consumerModule(t, root, "build_tags")
			runBoundary(t, dir, binary, mode, true, "./...")
		})
		for _, name := range []string{"valid", "rejected", "onlytag"} {
			t.Run(mode+"/"+name, func(t *testing.T) {
				dir := consumerModule(t, root, "build_tags")
				output := runWithBuildTags(t, dir, binary, mode, "./"+name+"/...", name == "valid")
				checkFixtureSnapshots(t, root, "build_tags/"+name, output)
			})
		}
	}
}

// runWithBuildTags selects tags through the documented CLI environment or vet flag.
func runWithBuildTags(t *testing.T, dir, binary, mode, target string, success bool) string {
	t.Helper()
	command := exec.Command(binary, target)
	flags := "-tags=integration"
	if mode == "vet" {
		command = exec.Command("go", "vet", "-vettool="+binary, "-tags=integration", target)
		flags = ""
	}
	command.Dir = dir
	command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS="+flags)
	output, err := command.CombinedOutput()
	if (err == nil) != success {
		t.Fatalf("%s: %v\n%s", command, err, output)
	}
	return string(output)
}
