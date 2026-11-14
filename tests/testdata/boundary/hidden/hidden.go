package hidden

import "github.com/lucasavila00/immutago/pkg/im"

type value struct{ im.Shared }

func Number() int { return 1 }
