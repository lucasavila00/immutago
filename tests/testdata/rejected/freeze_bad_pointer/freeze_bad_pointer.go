package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() { _ = im.FreezeSlice([]*int{new(int)}) }
