package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Count struct {
	im.Copied
	N int
}
type Reply struct {
	im.Local
	Count Count
}
type Failure = error
type Commands struct{}

func (Commands) Restart() error { return errors.New("restart failed") }

type Pool struct {
	supervise.Service[supervise.NoArgs]
	Commands
	count int
}

func (*Pool) Run(context.Context) {}
func (p *Pool) Add(name string) (Count, Failure) {
	if name == "" {
		return Count{}, fmt.Errorf("name: %w", errors.New("empty"))
	}
	p.count++
	return Count{N: p.count}, nil
}
func (p *Pool) Read() (Reply, error)            { return Reply{Count: Count{N: p.count}}, nil }
func (p *Pool) Named() (value Reply, err error) { value = Reply{}; return }
func (p *Pool) Forward() (Reply, error)         { return p.Read() }

// StateError deliberately keeps ordinary mutable state under the Go error contract.
type StateError struct{ Pool *Pool }

func (e *StateError) Error() string { return fmt.Sprintf("count: %d", e.Pool.count) }
func (p *Pool) Failure() error      { return &StateError{Pool: p} }
func Read() (Reply, error)          { return Reply{}, nil }
