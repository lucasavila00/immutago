package shadow

import (
	"context"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type error interface{ Error() string }
type Worker struct {
	supervise.Service[supervise.NoArgs]
}

func (*Worker) Run(context.Context) {}
func (*Worker) Restart() error      { return nil }
