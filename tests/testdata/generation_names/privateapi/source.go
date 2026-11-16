package privateapi

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import "github.com/lucasavila00/immutago/pkg/im"

type hidden struct {
	im.Frozen
	N int
}
type Public = hidden
