package aliasapi

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

type Point struct {
	im.Frozen
	N int
}

type RetryWrapped[A any] = supervise.Service[[]A]
type TerminalWrapped[A any] = supervise.TerminalService[[]A]
type RetryReordered[A, B any] = supervise.Service[[2]B]
type TerminalReordered[A, B any] = supervise.TerminalService[[2]B]
type RetryUnused[A any] = supervise.Service[int]
type TerminalUnused[A any] = supervise.TerminalService[int]
