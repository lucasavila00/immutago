package accepted

import "github.com/lucasavila00/immutago/pkg/im"

type SnapshotPoint struct {
	im.Frozen
	X int
}
type LocalPoint struct {
	im.Local
	X int
}
type PointGroup struct {
	im.Frozen
	SnapshotPoint
}
type Position = SnapshotPoint

func (p SnapshotPoint) XValue() int { return p.X }
func (p *SnapshotPoint) ReadX() int { return p.X }
func (p *LocalPoint) ReadX() int    { return p.X }
func makePoint() SnapshotPoint      { return SnapshotPoint{X: 3} }

func guide() {
	point := SnapshotPoint{X: 3}
	p := &SnapshotPoint{X: 3}
	point.XValue()
	group := PointGroup{SnapshotPoint: point}
	group.SnapshotPoint.XValue()
	p.ReadX()
	(*p).XValue()
	SnapshotPoint{X: 3}.XValue()
	(&SnapshotPoint{X: 3}).ReadX()
	makePoint().XValue()
	(p).ReadX()
	alias := Position{X: 3}
	alias.XValue()
	byValue := &LocalPoint{X: 3}
	byValue.ReadX()
}
