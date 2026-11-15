package hidden

import "github.com/lucasavila00/immutago/pkg/im"

type value struct{ im.Frozen }

func Number() int { return 1 }
