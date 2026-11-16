package api

import (
	"context"
	"sync/atomic"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Worker exposes a scalar method and a promoted method.
type Worker struct {
	supervise.Service[int]
	Methods
	Value int
	Next  *Worker
}

type Methods struct{}

func (Methods) Promoted() int           { return 1 }
func (*Worker) Run(context.Context)     {}
func (w *Worker) Count() int            { return w.Value }
func (w *Worker) Set(value int)         { w.Value = value }
func (w *Worker) Pair() (int, bool)     { return w.Value, true }
func Pointer() *Worker                  { return new(Worker) }
func Pointers() (*Worker, int)          { return Pointer(), 1 }
func New[T any]() *T                    { return new(T) }
func Keep[T any](value T) T             { return value }
func KeepPointer(value *Worker) *Worker { return value }

type Alias = Worker
type WorkerRef = supervise.Ref[Worker]
type DefinedRef WorkerRef
type Anonymous = struct {
	supervise.Service[int]
	Methods
}
type AnonymousRef = supervise.Ref[Anonymous]
type Box[T any] struct{ Value T }
type Holder struct{ Value *Worker }
type HolderAlias = Holder

func MakeHolder() Holder { return Holder{} }

func MakeAnon() struct{ Value *Worker } { return struct{ Value *Worker }{} }

func AnonPair() (struct{ Value *Worker }, int) { return MakeAnon(), 1 }

var Current = Pointer()

// Save keeps the service implementation's pointer operations ordinary.
func Save(value *Worker) func() *Worker {
	Current = value
	value.Next = value
	value.Value++
	return func() *Worker { return Current }
}

// PointerArgs is valid in the package that declares Worker.
type PointerArgs struct{ supervise.Service[*Worker] }

func (*PointerArgs) Run(context.Context) {}

// Control supplies one restart trigger for the runtime test.
type Control struct {
	Failure  chan struct{}
	Attempts atomic.Int64
}

type RestartWorker struct {
	supervise.Service[*Control]
	version int64
}

func (w *RestartWorker) Init() error {
	w.version = w.Args().Attempts.Add(1)
	return nil
}

func (w *RestartWorker) Run(ctx context.Context) {
	if w.version == 1 {
		select {
		case <-w.Args().Failure:
			panic("retry")
		case <-ctx.Done():
		}
		return
	}
	<-ctx.Done()
}

func (w *RestartWorker) Version() int64 { return w.version }
