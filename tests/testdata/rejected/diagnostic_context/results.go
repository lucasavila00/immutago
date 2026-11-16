package diagnosticcontext

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type SettingsReply struct {
	im.Local
	Value Settings
}

type Worker struct {
	supervise.Service[supervise.NoArgs]
}

func (w *Worker) Run(ctx context.Context) { <-ctx.Done() }

func (w *Worker) Read(value *Settings) Settings { return *value }
