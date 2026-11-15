package analyzer

import (
	"strings"
	"testing"

	"github.com/lucasavila00/immutago/internal/viewgen"
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
		{"missing directive", func(p *analysis.Pass) { p.Files[0].Comments = nil }, "exactly one package directive"},
		{"wrong command", func(p *analysis.Pass) { p.Files[0].Comments[0].List[0].Text += " -other" }, "package directive"},
		{"duplicate directive", func(p *analysis.Pass) {
			p.Files[0].Comments = append(p.Files[0].Comments, p.Files[0].Comments[0])
		}, "exactly one package directive"},
		{"missing output", func(p *analysis.Pass) { p.Files = p.Files[:1] }, "missing " + viewgen.Filename},
		{"stale output", func(p *analysis.Pass) {
			read := p.ReadFile
			p.ReadFile = func(path string) ([]byte, error) {
				content, err := read(path)
				return append(content, []byte("\n// changed\n")...), err
			}
		}, "is stale"},
		{"generated-only records", func(p *analysis.Pass) { p.Files = p.Files[1:] }, "no source marked records"},
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

// TestGeneratedDeclarationBounds excludes forms that cannot have a package-level API.
func TestGeneratedDeclarationBounds(t *testing.T) {
	for _, source := range []string{
		`func f(){type Data struct{im.Frozen;Number int};_=Data{}}`,
		`var data=struct{im.Frozen;Number int}{}`,
		`type Data=struct{im.Local;Number int}`,
	} {
		if got := checkSource(t, source); !strings.Contains(got, "package-level") {
			t.Fatal(got)
		}
	}
	for _, source := range []string{
		`type Reply struct{im.Local;Value im.StringView}`,
		"//line zz_immutago.gen.go:1\ntype Data struct{im.Frozen;Number int}",
	} {
		if got := checkSource(t, source); got != "" {
			t.Fatal(got)
		}
	}
}
