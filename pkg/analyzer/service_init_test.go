package analyzer

import (
	"strings"
	"testing"
)

const initDefinitions = `import("context"; "github.com/lucasavila00/immutago/pkg/supervise")
var _ im.Shared
type Initializer struct { supervise.Service[supervise.NoArgs]; ready chan struct{} }
func (*Initializer) Run(context.Context) {}
`

// TestServiceInitAccepted covers the exact optional hook and ordinary method-body forms.
func TestServiceInitAccepted(t *testing.T) {
	for _, body := range []string{
		``,
		`func (p *Initializer) Init() error { p.ready = make(chan struct{}, 1); return nil }`,
		`type InitError = error; func (*Initializer) Init() InitError { return nil }`,
		`func (p *Initializer) Init() error {
			_, err := supervise.Start[Child](p.Children(), supervise.NoArgs{}); return err
		}
		type Child struct { supervise.Service[supervise.NoArgs] }; func (*Child) Run(context.Context) {}`,
	} {
		t.Run(body, func(t *testing.T) {
			if got := checkSource(t, initDefinitions+body); got != "" {
				t.Fatal(got)
			}
		})
	}
}

// TestServiceInitDeclarationRejected checks the whole hook signature, including unused services.
func TestServiceInitDeclarationRejected(t *testing.T) {
	cases := []string{
		`func (Initializer) Init() error { return nil }`,
		`func (*Initializer) Init(int) error { return nil }`,
		`func (*Initializer) Init(...int) error { return nil }`,
		`func (*Initializer) Init() {}`,
		`func (*Initializer) Init() (error, bool) { return nil, false }`,
		`func (*Initializer) Init() interface{ Error() string } { return nil }`,
		`type InitError interface { Error() string }; func (*Initializer) Init() InitError { return nil }`,
	}
	for _, body := range cases {
		t.Run(body, func(t *testing.T) {
			got := checkSource(t, initDefinitions+body)
			if !strings.Contains(got, "a service must use the default Init or declare Init() error") {
				t.Fatalf("missing Init signature error: %s", got)
			}
		})
	}
	for _, fields := range []string{"Init int", "Other"} {
		t.Run(fields, func(t *testing.T) {
			source := `import("context"; "github.com/lucasavila00/immutago/pkg/supervise")
			var _ im.Shared
			type Other struct{}; func (*Other) Init() error { return nil }
			type Bad struct { supervise.Service[supervise.NoArgs]; ` + fields + ` }; func (*Bad) Run(context.Context) {}`
			if got := checkSource(t, source); !strings.Contains(got, "a service must use the default Init") {
				t.Fatalf("missing Init selection error: %s", got)
			}
		})
	}
}

// TestServiceInitUseRejected keeps both default and explicit hooks private to runtime calls.
func TestServiceInitUseRejected(t *testing.T) {
	for _, hook := range []string{"", `func (*Initializer) Init() error { return nil }`} {
		for _, body := range []string{
			`func (p *Initializer) again() { p.Init() }`,
			`func (p *Initializer) again() { f := p.Init; _ = f }`,
			`func f(ref supervise.Ref[Initializer]) { ref.Current().Init() }`,
			`func f(ref supervise.Ref[Initializer]) { f := ref.Current().Init; _ = f }`,
			`var f = (*Initializer).Init`,
			`func (p *Initializer) again() { p.Service.Init() }`,
			`var f = (*supervise.Service[supervise.NoArgs]).Init`,
		} {
			t.Run(hook+body, func(t *testing.T) {
				got := checkSource(t, initDefinitions+hook+"\n"+body)
				if !strings.Contains(got, "only the runtime can call a service Init method") {
					t.Fatalf("missing runtime-only Init diagnostic: %s", got)
				}
			})
		}
	}
}

// TestServiceStartBindings accepts only the listed two-result bindings.
func TestServiceStartBindings(t *testing.T) {
	for _, body := range []string{
		`ref, err := supervise.Start[Initializer](s, supervise.NoArgs{}); _ = ref; _ = err`,
		`_, err := supervise.Start[Initializer](s, supervise.NoArgs{}); _ = err`,
		`ref, _ := (supervise.Start[Initializer](s, supervise.NoArgs{})); _ = ref`,
		`var ref, err = supervise.Start[Initializer](s, supervise.NoArgs{}); _ = ref; _ = err`,
		`ref, err := supervise.Start[Initializer](s, supervise.NoArgs{})
other, err := supervise.Start[Initializer](s, supervise.NoArgs{})
		_ = ref; _ = other; _ = err`,
		`ref := supervise.Reserve[Initializer](); err := supervise.StartReserved(s, ref, supervise.NoArgs{}); _ = err`,
		`var ref supervise.Ref[Initializer]; var err error
		(ref), (err) = (supervise.Start[Initializer](s, supervise.NoArgs{})); _ = ref; _ = err`,
	} {
		t.Run(body, func(t *testing.T) {
			if got := checkSource(t, initDefinitions+`func f(s *supervise.Group) {`+body+`}`); got != "" {
				t.Fatal(got)
			}
		})
	}
	for _, source := range []string{
		`var ref, err = supervise.Start[Initializer](supervise.NewRoot(nil), supervise.NoArgs{})`,
		`func f(s *supervise.Group) { var holder struct { ref supervise.Ref[Initializer] }; var err error
		holder.ref, err = supervise.Start[Initializer](s, supervise.NoArgs{}); _ = err }`,
		`type Result struct { im.Shared }; func Start() (Result, error) { return Result{}, nil }
		func f() { value, err := Start(); _ = value; _ = err }`,
		`func f() { values := im.FreezeMap(map[int]int{}); var err bool
		value, err := values.Get(1); _ = value; _ = err }`,
	} {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, initDefinitions+source); got == "" {
				t.Fatal("unsupported tuple binding passed")
			}
		})
	}
}

// TestServiceScalarStartBindings preserves ordinary Go when a Start call has no protected arguments.
func TestServiceScalarStartBindings(t *testing.T) {
	source := `import("context"; "github.com/lucasavila00/immutago/pkg/supervise")
	var _ im.Shared
	type Counter struct { supervise.Service[int] }; func (*Counter) Run(context.Context) {}
	var globalRef, globalErr = supervise.Start[Counter](supervise.NewRoot(nil), 1)
	func f(s *supervise.Group) {
		var ref supervise.Ref[Counter]; var err error
		ref, err = supervise.Start[Counter](s, 1)
		var holder struct { ref supervise.Ref[Counter] }
		holder.ref, err = supervise.Start[Counter](s, 2)
		_ = ref; _ = err
	}`
	if got := checkSource(t, source); got != "" {
		t.Fatal(got)
	}
}

// TestPairDeclarationsRequireLocalScope checks blank names as well as named tuple destinations.
func TestPairDeclarationsRequireLocalScope(t *testing.T) {
	for _, call := range []string{
		`supervise.Start[Initializer](supervise.NewRoot(nil), supervise.NoArgs{})`,
		`frozen.Get(1)`,
	} {
		for _, names := range []string{"_, _", "value, _", "_, value"} {
			declaration := "var " + names + " = " + call
			t.Run(declaration, func(t *testing.T) {
				prefix := initDefinitions + `var frozen im.FrozenMap[int, int]; `
				if got := checkSource(t, prefix+declaration); got == "" {
					t.Fatal("package tuple declaration passed")
				}
				if names == "_, _" {
					if got := checkSource(t, prefix+`func f() { `+declaration+` }`); got != "" {
						t.Fatal(got)
					}
				}
			})
		}
	}
}
