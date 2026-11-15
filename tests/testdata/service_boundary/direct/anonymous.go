package direct

import "github.com/lucasavila00/immutago/pkg/supervise"

type MutableMessages struct{}

func (MutableMessages) Values() []int { return nil }

var anonymous supervise.Ref[struct {
	supervise.Service[supervise.NoArgs]
	MutableMessages
}]
