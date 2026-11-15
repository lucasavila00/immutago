package accepted

import "github.com/lucasavila00/immutago/pkg/im"

type SharedPoint struct {
	im.Shared
	X int
}
type ByValuePoint struct {
	im.ByValue
	X int
}
type PointGroup struct {
	im.Shared
	SharedPoint
}
type Position = SharedPoint

func (p SharedPoint) XValue() int  { return p.X }
func (p *SharedPoint) ReadX() int  { return p.X }
func (p *ByValuePoint) ReadX() int { return p.X }
func makePoint() SharedPoint       { return SharedPoint{X: 3} }

func guide() {
	point := SharedPoint{X: 3}
	p := &SharedPoint{X: 3}
	point.XValue()
	group := PointGroup{SharedPoint: point}
	group.SharedPoint.XValue()
	p.ReadX()
	(*p).XValue()
	SharedPoint{X: 3}.XValue()
	(&SharedPoint{X: 3}).ReadX()
	makePoint().XValue()
	(p).ReadX()
	alias := Position{X: 3}
	alias.XValue()
	byValue := &ByValuePoint{X: 3}
	byValue.ReadX()
}
