package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Frozen
	Value int
	Array [2]int
}

type Bad struct {
	im.Frozen
	Value any
}

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen
