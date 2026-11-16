package api

import "github.com/lucasavila00/immutago/pkg/im"

type Seconds struct{}
type Meter[Unit any] int
type Duration = Meter[Seconds]

func (m Meter[Unit]) Read() int                 { return int(m) }
func (m Meter[Unit]) Next() Meter[Unit]         { m++; return m }
func (m Meter[Unit]) Pair() (Meter[Unit], bool) { return m, true }
func (m *Meter[Unit]) Inc()                     { *m++ }

func (m Meter[Unit]) Add(values ...Meter[Unit]) Meter[Unit] {
	for _, value := range values {
		m += value
	}
	return m
}

type Range[Unit any] struct {
	im.Copied
	Low, High int
}

func (r Range[Unit]) Width() int { return r.High - r.Low }
func (r *Range[Unit]) Grow()     { r.High++ }
