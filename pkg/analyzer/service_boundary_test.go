package analyzer

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
)

// TestPrunedServiceBoundary checks service contracts without a direct im or supervise import.
func TestPrunedServiceBoundary(t *testing.T) {
	api := sourcePass(t, `import("context"; "github.com/lucasavila00/immutago/pkg/supervise")
var _ im.Shared
type Counter struct {supervise.Service[int]}
func (*Counter) Run(context.Context) {}
func (*Counter) Count() int {return 1}
type Handle = supervise.Ref[Counter]
func Get() Handle {var p Handle;return p}
type Scalar[T interface{~int | Handle}] = int
type Contract interface { Get() Handle }
type Phantom[T any] struct{}
func Make() Phantom[Handle] {return Phantom[Handle]{}}`).Pkg
	api.SetImports(nil)
	for _, declaration := range []string{
		`var ref = api.Get()`,
		`var callback = api.Get`,
		`var ref api.Handle`,
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
				t.Fatalf("service contract did not activate checking: %v", reports)
			}
		})
	}
}
