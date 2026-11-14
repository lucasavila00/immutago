package tests

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// testMainCase describes one test package and its expected diagnostics.
type testMainCase struct {
	pkg     string
	success bool
	want    []string
}

// TestGeneratedTestMain checks the generated runner and both real test variants.
func TestGeneratedTestMain(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "immutago")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	runCommand(t, root, true, "go", "build", "-o", binary, "./cmd/immutago")
	cache, err := exec.Command("go", "env", "GOCACHE").Output()
	if err != nil {
		t.Fatal(err)
	}
	cases := []testMainCase{
		{"valid", true, nil},
		{"withoutmain", true, nil},
		{"invalid", false, []string{"internal_test.go", "external_test.go", "writes to immutable objects"}},
		{"spoof.test", false, []string{"main.go", "unsupported reflect import"}},
	}
	for _, mode := range []string{"cli", "vet"} {
		t.Run(mode, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.pkg, func(t *testing.T) {
					checkTestMainPackage(t, root, binary, mode, strings.TrimSpace(string(cache)), tc)
				})
			}
		})
	}
}

// checkTestMainPackage checks one driver and source package.
func checkTestMainPackage(t *testing.T, root, binary, mode, cache string, tc testMainCase) {
	t.Helper()
	dir := consumerModule(t, root, "testmain")
	command, arguments := binary, []string{"./" + tc.pkg}
	if mode == "vet" {
		command, arguments = "go", []string{"vet", "-vettool=" + binary, "./" + tc.pkg}
	}
	output := runCommand(t, dir, tc.success, command, arguments...)
	checkFixtureSnapshots(t, root, "testmain/"+tc.pkg, output)
	for _, want := range tc.want {
		if !strings.Contains(output, want) {
			t.Errorf("missing %q in %s", want, output)
		}
	}
	if strings.Contains(output, cache) {
		t.Errorf("diagnostic named the Go build cache: %s", output)
	}
}
