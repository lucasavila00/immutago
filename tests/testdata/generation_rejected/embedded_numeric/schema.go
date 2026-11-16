package embedded_numeric

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import "github.com/lucasavila00/immutago/pkg/im"

type Counter int

func (c *Counter) Inc() { *c++ }

type Counted struct {
	im.Frozen
	Counter
}

func write(counted *Counted) { counted.Inc() }
