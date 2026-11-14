package analyzer

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
)

// TestPrunedProcessBoundary checks process contracts without a direct im or proc import.
func TestPrunedProcessBoundary(t *testing.T) {
	api := sourcePass(t, `import("context"; "github.com/lucasavila00/immutago/pkg/proc")
var _ im.Shared
type Counter struct {proc.Process[int]}
func (*Counter) Run(context.Context) {}
func (*Counter) Count() int {return 1}
type Handle = proc.PID[Counter]
func Get() Handle {var p Handle;return p}
type Scalar[T interface{~int | Handle}] = int
type Contract interface { Get() Handle }
type Phantom[T any] struct{}
func Make() Phantom[Handle] {return Phantom[Handle]{}}`).Pkg
	api.SetImports(nil)
	for _, declaration := range []string{
		`var pid = api.Get()`,
		`var callback = api.Get`,
		`var pid api.Handle`,
		`var value api.Scalar[int]`,
		`var service api.Contract`,
		`var value = api.Make()`,
	} {
		t.Run(declaration, func(t *testing.T) {
			pass := boundaryPass(t, dependencyImporter{"sample": api},
				`package consumer;import api "sample";func missingBody();`+declaration)
			if usesImmutable(pass.Pkg) {
				t.Fatal("consumer has a direct library import")
			}
			var reports []string
			pass.Report = func(d analysis.Diagnostic) { reports = append(reports, d.Message) }
			if _, err := run(pass); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(strings.Join(reports, "\n"), "unsupported function without a Go body") {
				t.Fatalf("process contract did not activate checking: %v", reports)
			}
		})
	}
}
