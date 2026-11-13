package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Child struct {im.Marker; X int}
type Good struct {im.Marker; X,Y int; Child Child; Array [2]int}
func read(p *Good) int {return p.X}

func bad(){it:=im.NewSlice(Good{}).All();_=it}
