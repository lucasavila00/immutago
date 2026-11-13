package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() { _ = im.FreezeMap(map[*int]int{new(int): 1}) }
