package service

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type ReadyService struct{ supervise.Service[int] }

func (*ReadyService) Init() error             { return nil }
func (*ReadyService) Run(ctx context.Context) { <-ctx.Done() }

func (p *ReadyService) initializeAgain() {
	p.Init()
	callback := p.Init
	_ = callback
}

func callInit(ref supervise.Ref[ReadyService]) {
	ref.Current().Init()
}

type WrongInit struct{ supervise.Service[int] }

func (*WrongInit) Init()                   {}
func (*WrongInit) Run(ctx context.Context) { <-ctx.Done() }

type ValueInit struct{ supervise.Service[int] }

func (ValueInit) Init() error              { return nil }
func (*ValueInit) Run(ctx context.Context) { <-ctx.Done() }

func replaceRef(root *supervise.Group) {
	var holder struct{ Ref supervise.Ref[DefaultInit] }
	var err error
	holder.Ref, err = supervise.Start[DefaultInit](root, supervise.NoArgs{})
	_ = err
}

// DefaultInit uses the embedded no-op initializer.
type DefaultInit struct {
	supervise.Service[supervise.NoArgs]
}

func (*DefaultInit) Run(ctx context.Context) { <-ctx.Done() }

func (p *DefaultInit) initializeAgain() {
	p.Init()
	callback := p.Init
	_ = callback
	p.Service.Init()
}

var customInitializer = (*ReadyService).Init
var defaultInitializer = (*supervise.Service[supervise.NoArgs]).Init

type InitFailure interface{ Error() string }
type WrongError struct{ supervise.Service[int] }

func (*WrongError) Init() InitFailure   { return nil }
func (*WrongError) Run(context.Context) {}

type InitParameters struct{ supervise.Service[int] }

func (*InitParameters) Init(...int) error   { return nil }
func (*InitParameters) Run(context.Context) {}

type InitHelper struct{}

func (*InitHelper) Init() error { return nil }

type PromotedInit struct {
	supervise.Service[int]
	InitHelper
}

func (*PromotedInit) Run(context.Context) {}

type InitField struct {
	supervise.Service[int]
	Init int
}

func (*InitField) Run(context.Context) {}

var _, _ = supervise.Start[DefaultInit](supervise.NewRoot(nil), supervise.NoArgs{})
