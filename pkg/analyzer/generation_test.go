package analyzer

import (
	"go/ast"
	"go/parser"
	"go/types"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasavila00/immutago/internal/projectiongen"
	"golang.org/x/tools/go/analysis"
)

// TestRequiredGeneration checks the canonical directive and generated source.
func TestRequiredGeneration(t *testing.T) {
	cases := []struct {
		name   string
		change func(*analysis.Pass)
		want   string
	}{
		{"valid", func(*analysis.Pass) {}, ""},
		{"ordinary generator", func(p *analysis.Pass) {
			p.Files[0].Comments = append(p.Files[0].Comments, &ast.CommentGroup{List: []*ast.Comment{
				{Slash: p.Files[0].Comments[0].Pos(), Text: "//go:generate echo immutago"},
			}})
		}, ""},
		{"missing directive", func(p *analysis.Pass) { p.Files[0].Comments = nil }, "exactly one package directive"},
		{"test-only directive", func(p *analysis.Pass) {
			p.Files[0].Comments = nil
			file, err := parser.ParseFile(p.Fset, "directive_test.go", projectiongen.Directive+"\npackage sample\n", parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			p.Files = append(p.Files, file)
		}, "exactly one package directive"},
		{"wrong command", func(p *analysis.Pass) { p.Files[0].Comments[0].List[0].Text += " -other" }, "package directive"},
		{"duplicate directive", func(p *analysis.Pass) {
			p.Files[0].Comments = append(p.Files[0].Comments, p.Files[0].Comments[0])
		}, "exactly one package directive"},
		{"missing output", func(p *analysis.Pass) { p.Files = p.Files[:1] }, "missing " + projectiongen.Filename},
		{"stale output", func(p *analysis.Pass) {
			read := p.ReadFile
			p.ReadFile = func(path string) ([]byte, error) {
				content, err := read(path)
				return append(content, []byte("\n// changed\n")...), err
			}
		}, "is stale"},
		{"generated-only records", func(p *analysis.Pass) { p.Files = p.Files[1:] }, "no source Frozen records"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			pass := sourcePass(t, `type Data struct{im.Frozen;Number int;Name string}`)
			test.change(pass)
			var reports []string
			pass.Report = func(d analysis.Diagnostic) { reports = append(reports, d.Message) }
			if _, err := run(pass); err != nil {
				t.Fatal(err)
			}
			got := strings.Join(reports, "\n")
			if (got == "") != (test.want == "") || !strings.Contains(got, test.want) {
				t.Fatalf("got %q; want %q", got, test.want)
			}
		})
	}
}

// TestGeneratedBridgeValidation grants no raw access after output or its physical name changes.
func TestGeneratedBridgeValidation(t *testing.T) {
	const declarations = `type Child struct{im.Frozen;Number int}
type Data struct{im.Frozen;Name string;Numbers [2]int;Number int;Child Child}`
	for _, body := range []string{
		`func raw(v DataProjection)string{return v.immutagoProjection.Read().Name}`,
		`func raw(v DataProjection)[]int{return v.immutagoProjection.Read().Numbers[:]}`,
		`func raw(v DataProjection)*int{return &v.immutagoProjection.Read().Number}`,
		`func raw(v DataProjection)*Child{return &v.immutagoProjection.Read().Child}`,
		`func raw(v DataProjection){v.immutagoProjection.Read().Number=1}`,
	} {
		t.Run(body, func(t *testing.T) {
			pass := sourcePass(t, declarations)
			generated, err := pass.ReadFile(projectiongen.Filename)
			if err != nil {
				t.Fatal(err)
			}
			replaceGeneratedSource(t, pass, projectiongen.Filename, string(generated)+"\n"+body)
			var reports []string
			pass.Report = func(d analysis.Diagnostic) { reports = append(reports, d.Message) }
			if _, err := run(pass); err != nil {
				t.Fatal(err)
			}
			got := strings.Join(reports, "\n")
			if !strings.Contains(got, "is stale") || !strings.Contains(got, "reserved for verified generated") {
				t.Fatal(got)
			}
			if strings.Contains(body, "Number=1") && !strings.Contains(got, "writes to immutable objects") {
				t.Fatal(got)
			}
		})
	}
	for _, prefix := range []string{"", "//line zz_immutago.gen.go:1\n"} {
		pass := sourcePass(t, declarations)
		generated, err := pass.ReadFile(projectiongen.Filename)
		if err != nil {
			t.Fatal(err)
		}
		replaceGeneratedSource(t, pass, "manual.go", prefix+string(generated))
		var reports []string
		pass.Report = func(d analysis.Diagnostic) { reports = append(reports, d.Message) }
		if _, err := run(pass); err != nil {
			t.Fatal(err)
		}
		if got := strings.Join(reports, "\n"); !strings.Contains(got, "reserved for verified generated") {
			t.Fatal(got)
		}
	}
}

// replaceGeneratedSource compiles the changed bytes so the source check sees the actual program.
func replaceGeneratedSource(t *testing.T, pass *analysis.Pass, name, content string) {
	t.Helper()
	file, err := parser.ParseFile(pass.Fset, name, content, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if len(pass.Files) == 1 {
		pass.Files = append(pass.Files, file)
	} else {
		pass.Files[1] = file
	}
	config := types.Config{Importer: testImporter{library: pass.Pkg.Imports()[0]}}
	pass.TypesInfo = sourceTypeInfo()
	pass.Pkg, err = config.Check(pass.Pkg.Path(), pass.Fset, pass.Files, pass.TypesInfo)
	if err != nil {
		t.Fatal(err)
	}
	read := pass.ReadFile
	pass.ReadFile = func(path string) ([]byte, error) {
		if filepath.Base(path) == name {
			return []byte(content), nil
		}
		return read(path)
	}
}

// TestGeneratedDeclarationBounds excludes forms that cannot have a package-level API.
func TestGeneratedDeclarationBounds(t *testing.T) {
	for _, source := range []string{
		`func f(){type Data struct{im.Frozen;Number int};_=Data{}}`,
		`var data=struct{im.Frozen;Number int}{}`,
		`type Data=struct{im.Frozen;Number int}`,
	} {
		if got := checkSource(t, source); !strings.Contains(got, "package-level") {
			t.Fatal(got)
		}
	}
	for _, source := range []string{
		`type Reply struct{im.Local;Value im.StringProjection}`,
		"//line zz_immutago.gen.go:1\ntype Data struct{im.Frozen;Number int}",
	} {
		if got := checkSource(t, source); got != "" {
			t.Fatal(got)
		}
	}
}

// TestLocalRecordsNeedNoGeneration keeps local composition separate from stored-data accessors.
func TestLocalRecordsNeedNoGeneration(t *testing.T) {
	for _, source := range []string{
		`type Reply struct{im.Local;Text im.StringProjection}`,
		`func f(){type Reply struct{im.Local;Text im.StringProjection};_=Reply{}}`,
		`func f(){_=struct{im.Local;Text im.StringProjection}{}}`,
	} {
		pass := sourcePass(t, source)
		pass.Files[0].Comments = nil
		pass.Report = func(d analysis.Diagnostic) { t.Fatal(d.Message) }
		if _, err := run(pass); err != nil {
			t.Fatal(err)
		}
	}
}

// TestLocalGeneratedFileCannotRead rejects a generated-file name without a Frozen schema.
func TestLocalGeneratedFileCannotRead(t *testing.T) {
	pass := sourcePass(t, `type Reply struct{im.Local;Text im.StringProjection}`)
	replaceGeneratedSource(t, pass, projectiongen.Filename, `package sample
import "github.com/lucasavila00/immutago/pkg/im"
func raw(value im.Projection[string])string{return value.Read()}`)
	var reports []string
	pass.Report = func(d analysis.Diagnostic) { reports = append(reports, d.Message) }
	if _, err := run(pass); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(reports, "\n")
	if !strings.Contains(got, "no source Frozen records") || !strings.Contains(got, "reserved for verified generated") {
		t.Fatal(got)
	}
}
