package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Child struct {im.Marker; X int}
type Good struct {im.Marker; X,Y int; Child Child; Array [2]int}
func read(p *Good) int {return p.X}

type Row struct{im.Marker; Values [2]Child}
func bad(v Row){p:=&v.Values[0];v.Values[1].X=2;_=p.X}
