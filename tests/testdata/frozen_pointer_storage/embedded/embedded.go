//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package embedded

import "github.com/lucasavila00/immutago/pkg/im"

type Marker = im.Frozen
type Count int

func (c *Count) Inc() { *c++ }

type Stats struct {
	im.Copied
	N int
}

func (s *Stats) Inc() { s.N++ }

type Numeric struct {
	Marker
	Count
}

type Record struct {
	Marker
	Stats
}

type Pointer struct {
	Marker
	*Count
}

func bump[T interface{ Inc() }](value T) { value.Inc() }

func Dispatch(p *Numeric, r *Record, q *Pointer, callback func(interface{ Inc() })) {
	var value interface{ Inc() } = p
	value.Inc()
	bump(r)
	callback(q)
}
