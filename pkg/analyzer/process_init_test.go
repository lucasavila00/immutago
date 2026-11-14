package analyzer

import (
	"strings"
	"testing"
)

const initDefinitions = `import("context"; "github.com/lucasavila00/immutago/pkg/proc")
var _ im.Shared
type Initializer struct { proc.Process[proc.None]; ready chan struct{} }
func (*Initializer) Run(context.Context) {}
`

// TestProcessInitAccepted covers the exact optional hook and ordinary method-body forms.
func TestProcessInitAccepted(t *testing.T) {
	for _, body := range []string{
		``,
		`func (p *Initializer) Init() error { p.ready = make(chan struct{}, 1); return nil }`,
		`type InitError = error; func (*Initializer) Init() InitError { return nil }`,
		`func (p *Initializer) Init() error {
			_, err := proc.Start[Child](p.Supervisor(), proc.None{}); return err
		}
		type Child struct { proc.Process[proc.None] }; func (*Child) Run(context.Context) {}`,
	} {
		t.Run(body, func(t *testing.T) {
			if got := checkSource(t, initDefinitions+body); got != "" {
				t.Fatal(got)
			}
		})
	}
}

// TestProcessInitDeclarationRejected checks the whole hook signature, including unused processes.
func TestProcessInitDeclarationRejected(t *testing.T) {
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
			if !strings.Contains(got, "a process must use the default Init or declare Init() error") {
				t.Fatalf("missing Init signature error: %s", got)
			}
		})
	}
	for _, fields := range []string{"Init int", "Other"} {
		t.Run(fields, func(t *testing.T) {
			source := `import("context"; "github.com/lucasavila00/immutago/pkg/proc")
			var _ im.Shared
			type Other struct{}; func (*Other) Init() error { return nil }
			type Bad struct { proc.Process[proc.None]; ` + fields + ` }; func (*Bad) Run(context.Context) {}`
			if got := checkSource(t, source); !strings.Contains(got, "a process must use the default Init") {
				t.Fatalf("missing Init selection error: %s", got)
			}
		})
	}
}

// TestProcessInitUseRejected keeps both default and explicit hooks private to runtime calls.
func TestProcessInitUseRejected(t *testing.T) {
	for _, hook := range []string{"", `func (*Initializer) Init() error { return nil }`} {
		for _, body := range []string{
			`func (p *Initializer) again() { p.Init() }`,
			`func (p *Initializer) again() { f := p.Init; _ = f }`,
			`func f(pid proc.PID[Initializer]) { pid.Proc().Init() }`,
			`func f(pid proc.PID[Initializer]) { f := pid.Proc().Init; _ = f }`,
			`var f = (*Initializer).Init`,
			`func (p *Initializer) again() { p.Process.Init() }`,
			`var f = (*proc.Process[proc.None]).Init`,
		} {
			t.Run(hook+body, func(t *testing.T) {
				if got := checkSource(t, initDefinitions+hook+"\n"+body); got == "" {
					t.Fatal("runtime-only Init use passed")
				}
			})
		}
	}
}

// TestProcessStartBindings accepts only the listed two-result bindings.
func TestProcessStartBindings(t *testing.T) {
	for _, body := range []string{
		`pid, err := proc.Start[Initializer](s, proc.None{}); _ = pid; _ = err`,
		`_, err := proc.Start[Initializer](s, proc.None{}); _ = err`,
		`pid, _ := (proc.Start[Initializer](s, proc.None{})); _ = pid`,
		`var pid, err = proc.Start[Initializer](s, proc.None{}); _ = pid; _ = err`,
		`pid, err := proc.Start[Initializer](s, proc.None{}); other, err := proc.Start[Initializer](s, proc.None{})
		_ = pid; _ = other; _ = err`,
		`pid := proc.Reserve[Initializer](); err := proc.StartReserved(s, pid, proc.None{}); _ = err`,
		`var pid proc.PID[Initializer]; var err error
		pid, err = proc.Start[Initializer](s, proc.None{}); _ = pid; _ = err`,
	} {
		t.Run(body, func(t *testing.T) {
			if got := checkSource(t, initDefinitions+`func f(s *proc.Supervisor) {`+body+`}`); got != "" {
				t.Fatal(got)
			}
		})
	}
	for _, source := range []string{
		`var pid, err = proc.Start[Initializer](proc.NewRoot(nil), proc.None{})`,
		`func f(s *proc.Supervisor) { var holder struct { pid proc.PID[Initializer] }; var err error
		holder.pid, err = proc.Start[Initializer](s, proc.None{}); _ = err }`,
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

// TestProcessScalarStartBindings preserves ordinary Go when a Start call has no protected arguments.
func TestProcessScalarStartBindings(t *testing.T) {
	source := `import("context"; "github.com/lucasavila00/immutago/pkg/proc")
	var _ im.Shared
	type Counter struct { proc.Process[int] }; func (*Counter) Run(context.Context) {}
	var globalPID, globalErr = proc.Start[Counter](proc.NewRoot(nil), 1)
	func f(s *proc.Supervisor) {
		var pid proc.PID[Counter]; var err error
		pid, err = proc.Start[Counter](s, 1)
		var holder struct { pid proc.PID[Counter] }
		holder.pid, err = proc.Start[Counter](s, 2)
		_ = pid; _ = err
	}`
	if got := checkSource(t, source); got != "" {
		t.Fatal(got)
	}
}
