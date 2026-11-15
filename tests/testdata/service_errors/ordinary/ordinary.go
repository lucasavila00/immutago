package ordinary

import (
	"context"
	"errors"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Worker struct {
	supervise.Service[supervise.NoArgs]
}

func (*Worker) Run(context.Context)       {}
func (*Worker) Restart() error            { return errors.New("failed") }
func (*Worker) Read() (int, error)        { return 1, nil }
func Use(ref supervise.Ref[Worker]) error { _, err := ref.Borrow().Read(); return err }
