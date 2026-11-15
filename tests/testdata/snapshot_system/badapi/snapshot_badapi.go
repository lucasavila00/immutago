package badapi

import "github.com/lucasavila00/immutago/pkg/im"

type Result struct {
	im.Local
	Name string
}

func Read() Result { return Result{} }
