package contracts

import (
	"example.com/consumer/api"
	"example.com/consumer/mixedapi"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Getter func() mixedapi.Pointer
type Reader interface{ Read() mixedapi.Pointer }

func mixed() {
	read := mixedapi.Mixed
	_, _ = read()
}

type LocalHolder struct {
	im.Local
	Worker *api.Worker
}

var cell im.Cell[*api.Worker]

func freeze(worker *api.Worker) {
	values := []*api.Worker{worker}
	_ = im.FreezeSlice(values)
}
