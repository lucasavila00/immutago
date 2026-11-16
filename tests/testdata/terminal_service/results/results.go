package results

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Base = supervise.TerminalService[supervise.NoArgs]
type Worker struct{ Base }

func (*Worker) Run(ctx context.Context)    { <-ctx.Done() }
func (*Worker) Text() string               { return "native string" }
func (*Worker) Mixed() (int, []int, error) { return 1, nil, nil }
func (*Worker) Pointer() *int              { return nil }
func (*Worker) Marker() im.Local           { return im.Local{} }

type Error error

func (*Worker) NamedError() Error { return nil }

type Nested struct{ Worker }

type Defined Worker

func (*Defined) Run(ctx context.Context) { <-ctx.Done() }
func (*Defined) Value() map[int]int      { return nil }
