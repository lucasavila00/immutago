package mixedapi

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type View = im.StringProjection

type Worker struct{ supervise.Service[int] }

func (*Worker) Run(context.Context) {}
func (*Worker) Count() int          { return 1 }

// Mixed has an im.Local result that requires a direct typed call.
func Mixed() (*Worker, View) {
	return new(Worker), im.ProjectString("ready")
}

type LocalRecord struct {
	im.Local
	N int
}

type Pointer = *LocalRecord
