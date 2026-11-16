package badapi

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Pool struct {
	supervise.Service[supervise.NoArgs]
}
type DefinedError error
type Unmarked struct {
	Count int
	Found bool
}
type Names [2]int

func (*Pool) First() (string, int, bool)           { return "", 0, false }
func (*Pool) Middle() (api.View, []int, error)     { return api.View{}, nil, nil }
func (*Pool) Last() (error, bool, map[int]int)     { return nil, false, nil }
func (*Pool) Pointer() (bool, *api.View)           { return false, nil }
func (*Pool) Record() (bool, Unmarked)             { return false, Unmarked{} }
func (*Pool) ErrorType() (int, DefinedError, bool) { return 0, nil, false }
func (*Pool) Marker() (int, im.Local, bool)        { return 0, im.Local{}, false }
func (*Pool) Array() (bool, [2]Names)              { return false, [2]Names{} }
func (*Pool) Frozen() (bool, im.FrozenString)      { return false, im.FreezeString("") }
func (*Pool) Nested() (bool, [2][1]*api.View)      { return false, [2][1]*api.View{} }
func (*Pool) FrozenRecord() (api.Frozen, bool)     { return api.Frozen{}, false }
