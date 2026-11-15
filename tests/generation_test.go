package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/lucasavila00/immutago/internal/projectiongen"
)

// TestGenerationContract runs the required command in an independent consumer module.
func TestGenerationContract(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "immutago")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	runCommand(t, root, true, "go", "build", "-o", binary, "./cmd/immutago")
	dir := consumerModule(t, root, "generation_contract")
	check := func(name string, success bool) {
		t.Helper()
		output := runCommand(t, dir, success, binary, "./...")
		checkSnapshot(t, filepath.Join(root, "tests/testdata/generation_contract/cases", name, "output.txt"),
			normalizeOutput(output))
	}
	check("missing", false)
	runCommand(t, dir, true, "go", "get", "-tool", "github.com/lucasavila00/immutago/cmd/immutago-gen")
	runCommand(t, dir, true, "go", "generate", "./...")
	check("generated", true)
	runCommand(t, dir, true, "go", "vet", "-vettool="+binary, "./...")
	sourcePath := filepath.Join(dir, "schema.go")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, sourcePath, string(source)+"\nfunc raw(value im.Projection[int]) int { return value.Read() }\n")
	check("handwritten_read", false)
	changed := strings.Replace(string(source), "Number int", "Number int; Ready bool", 1)
	writeFile(t, sourcePath, changed)
	check("stale", false)
	runCommand(t, dir, true, "go", "generate", "./...")
	check("regenerated", true)
	outputPath := filepath.Join(dir, projectiongen.Filename)
	generated, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ name, source string }{
		{"generated_write", `func illegal(value DataProjection) { value.immutagoProjection.Read().Number = 1 }`},
		{"generated_string", `func illegal(value DataProjection) string { return value.immutagoProjection.Read().Text }`},
		{"generated_slice", `func illegal(value DataProjection) []int { return value.immutagoProjection.Read().Values[:] }`},
		{"generated_address", `func illegal(value DataProjection) *int { return &value.immutagoProjection.Read().Number }`},
	} {
		writeFile(t, outputPath, string(generated)+"\n"+test.source+"\n")
		check(test.name, false)
	}
	runCommand(t, dir, true, "go", "generate", "./...")
	writeFile(t, sourcePath, strings.Replace(changed, projectiongen.Directive, "// Generate the readers.", 1))
	check("directive", false)
	writeFile(t, sourcePath, "package contract\nimport \"github.com/lucasavila00/immutago/pkg/im\"\n"+
		"type Reply struct { im.Local; Text im.StringProjection; Number int }\n")
	if err := os.Remove(outputPath); err != nil {
		t.Fatal(err)
	}
	check("local_only", true)
}

// TestGenerationScopes rejects schemas that cannot have package-level generated readers.
func TestGenerationScopes(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "immutago")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	runCommand(t, root, true, "go", "build", "-o", binary, "./cmd/immutago")
	for _, test := range []struct{ name, file, source string }{
		{"local", "schema.go", "func f(){type Data struct{im.Frozen;Number int};_=Data{}}"},
		{"anonymous", "schema.go", "var value=struct{im.Frozen;Number int}{}"},
		{"test", "schema_test.go", "type Data struct{im.Frozen;Number int}"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := consumerModule(t, root, "generation_contract")
			writeFile(t, filepath.Join(dir, "schema.go"), "package contract\n")
			writeFile(t, filepath.Join(dir, test.file), "package contract\n"+
				"import \"github.com/lucasavila00/immutago/pkg/im\"\n"+test.source+"\n")
			output := runCommand(t, dir, false, binary, "./...")
			checkSnapshot(t, filepath.Join(root, "tests/testdata/generation_contract/cases", test.name, "output.txt"),
				normalizeOutput(output))
		})
	}
}
