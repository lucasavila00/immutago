package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Child struct {im.Shared; X int}
type Good struct {im.Shared; X,Y int; Child Child; Array [2]int}
func read(p *Good) int {return p.X}

var Global Good
func bad(){_= &Global}
