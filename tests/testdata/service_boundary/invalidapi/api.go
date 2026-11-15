package invalidapi

import "github.com/lucasavila00/immutago/pkg/supervise"

type Counter struct{ supervise.Service[int] }

func (*Counter) Values() []int { return nil }
func Count() int               { return 1 }
