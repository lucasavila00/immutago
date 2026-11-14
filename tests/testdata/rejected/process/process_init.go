package process

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/proc"
)

type ReadyProcess struct{ proc.Process[int] }

func (*ReadyProcess) Init() error             { return nil }
func (*ReadyProcess) Run(ctx context.Context) { <-ctx.Done() }

func (p *ReadyProcess) initializeAgain() {
	p.Init()
	callback := p.Init
	_ = callback
}

func callInit(pid proc.PID[ReadyProcess]) {
	pid.Proc().Init()
}

type WrongInit struct{ proc.Process[int] }

func (*WrongInit) Init()                   {}
func (*WrongInit) Run(ctx context.Context) { <-ctx.Done() }

type ValueInit struct{ proc.Process[int] }

func (ValueInit) Init() error              { return nil }
func (*ValueInit) Run(ctx context.Context) { <-ctx.Done() }

func replacePID(root *proc.Supervisor) {
	var holder struct{ PID proc.PID[DefaultInit] }
	var err error
	holder.PID, err = proc.Start[DefaultInit](root, proc.None{})
	_ = err
}

// DefaultInit uses the embedded no-op initializer.
type DefaultInit struct{ proc.Process[proc.None] }

func (*DefaultInit) Run(ctx context.Context) { <-ctx.Done() }

func (p *DefaultInit) initializeAgain() {
	p.Init()
	callback := p.Init
	_ = callback
	p.Process.Init()
}

var customInitializer = (*ReadyProcess).Init
var defaultInitializer = (*proc.Process[proc.None]).Init

type InitFailure interface{ Error() string }
type WrongError struct{ proc.Process[int] }

func (*WrongError) Init() InitFailure   { return nil }
func (*WrongError) Run(context.Context) {}

type InitParameters struct{ proc.Process[int] }

func (*InitParameters) Init(...int) error   { return nil }
func (*InitParameters) Run(context.Context) {}

type InitHelper struct{}

func (*InitHelper) Init() error { return nil }

type PromotedInit struct {
	proc.Process[int]
	InitHelper
}

func (*PromotedInit) Run(context.Context) {}

type InitField struct {
	proc.Process[int]
	Init int
}

func (*InitField) Run(context.Context) {}

var _, _ = proc.Start[DefaultInit](proc.NewRoot(nil), proc.None{})
