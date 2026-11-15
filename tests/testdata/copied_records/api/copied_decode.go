package api

import (
	"strconv"

	"github.com/lucasavila00/immutago/pkg/im"
)

type Ratio struct {
	im.Copied
	value float64
}

func (r Ratio) Value() float64 { return r.value }
func (r Ratio) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatFloat(r.value, 'g', -1, 64)), nil
}
func (r *Ratio) UnmarshalJSON(data []byte) error {
	value, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return err
	}
	r.value = value
	return nil
}
func (r *Ratio) UnmarshalText(data []byte) error { return r.UnmarshalJSON(data) }
func (p *Point) Set(x int)                       { p.X = x }
func (p Point) Values() (int, int)               { return p.X, p.Y }
func (s *Score) Set(value float64)               { s.value = value }
