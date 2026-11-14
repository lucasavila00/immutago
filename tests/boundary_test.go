package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestDependencyBoundary checks ordinary dependencies and blocked protected transport in both drivers.
func TestDependencyBoundary(t *testing.T) {
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
		t.Run(mode, func(t *testing.T) { checkDependencyBoundary(t, root, binary, mode) })
	}
}

// checkDependencyBoundary uses independent modules so vet must obtain fresh dependency facts.
func checkDependencyBoundary(t *testing.T, root, binary, mode string) {
	t.Helper()
	checkCheckedBoundary(t, root, binary, mode)
	for _, target := range []string{"./...", "./loader", "./scalarunsafe"} {
		dir := consumerModule(t, root, "boundary")
		runBoundary(t, dir, binary, mode, true, target)
	}
	checkBoundaryFixtures(t, root, binary, mode, "boundary_rejected", "loader")
	checkBoundaryFixtures(t, root, binary, mode, "boundary_contracts", "contract")
}

// checkBoundaryFixtures adds each rejected source to its own independent consumer module.
func checkBoundaryFixtures(t *testing.T, root, binary, mode, fixture, target string) {
	t.Helper()
	fixtures := filepath.Join(root, "tests", "testdata", fixture)
	entries, err := os.ReadDir(fixtures)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			dir := consumerModule(t, root, "boundary")
			source, err := os.ReadFile(filepath.Join(fixtures, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(dir, target), 0755); err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(dir, target, entry.Name()), string(source))
			output := runBoundary(t, dir, binary, mode, false, "./"+target)
			if fixture == "boundary_contracts" && !strings.Contains(output, "unsupported unsafe import") {
				t.Fatalf("local contract did not activate checking: %s", output)
			}
			checkSnapshot(t, filepath.Join(fixtures, strings.TrimSuffix(entry.Name(), ".go")+".txt"), normalizeOutput(output))
		})
	}
}

// checkCheckedBoundary keeps unsafe and reflection values forbidden inside the checked import closure.
func checkCheckedBoundary(t *testing.T, root, binary, mode string) {
	t.Helper()
	for _, name := range []string{"unsafe_import", "raw_unsafe_import", "indirect_unsafe", "indirect_reflection",
		"unsafe_helper", "reflect_helper"} {
		t.Run(name, func(t *testing.T) {
			dir := consumerModule(t, root, "rejected")
			ordinary := name == "unsafe_helper" || name == "reflect_helper"
			output := runBoundary(t, dir, binary, mode, ordinary, "./"+name)
			checkFixtureSnapshots(t, root, "rejected/"+name, output)
		})
	}
}

// runBoundary selects the command driver without changing the requested package set.
func runBoundary(t *testing.T, dir, binary, mode string, success bool, target string) string {
	t.Helper()
	if mode == "vet" {
		return runCommand(t, dir, success, "go", "vet", "-vettool="+binary, target)
	}
	return runCommand(t, dir, success, binary, target)
}
