package signatures

import (
	"context"
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type OtherError error
type Pool struct {
	supervise.Service[supervise.NoArgs]
}

func (*Pool) Run(context.Context)                    {}
func (*Pool) Pair() (int, bool)                      { return 0, false }
func (*Pool) Reverse() (error, api.Reply)            { return nil, api.Reply{} }
func (*Pool) Three() (api.Reply, error, error)       { return api.Reply{}, nil, nil }
func (*Pool) Errors() (error, error)                 { return nil, nil }
func (*Pool) Text() (string, error)                  { return "", nil }
func (*Pool) Pointer() (*api.Reply, error)           { return nil, nil }
func (*Pool) Defined() OtherError                    { return nil }
func (*Pool) Interface() interface{ Error() string } { return nil }
