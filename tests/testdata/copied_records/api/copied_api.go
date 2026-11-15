package api

import "github.com/lucasavila00/immutago/pkg/im"

type Score struct {
	im.Copied
	value float64
	flags uint8
}

func NewScore(value float64) Score { return Score{value: value, flags: 1} }
func (s Score) Value() float64     { return s.value }

type Marker = im.Copied
type Point struct {
	Marker
	X int
	Y int
}

type Alias = Point
type Named Point
type Counts uint32
type Summary struct {
	im.Copied
	Score  Score
	Points [2][2]Alias
	Counts [2]Counts
	OK     bool
}

func (p Point) Sum() int { return p.X + p.Y }

func Values(values ...Point) int {
	total := 0
	for _, value := range values {
		total += value.X
	}
	return total
}
