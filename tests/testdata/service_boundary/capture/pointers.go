package capture

import (
	"sync/atomic"

	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Pointer = *api.Worker
type DefinedPointer *api.Worker
type Holder struct{ Value Pointer }
type Nested struct{ Values [1][2]Pointer }
type Generic[T any] struct{ Value Pointer }
type Atomic struct{ Value atomic.Pointer[api.Worker] }

var zero Pointer
var boxed any = api.Current
var array [1]Pointer
var values []Pointer
var mapping map[int]Pointer
var channel chan Pointer

func clearGlobal() { api.Current = nil }

func clearField() {
	value := api.Worker{}
	value.Next = nil
}

func returnNil() Pointer                       { return nil }
func returnPointer(value Pointer) Pointer      { return value }
func returnTuple() (int, Pointer)              { return 1, api.Pointer() }
func returnIndirect(ref api.WorkerRef) Pointer { return api.Keep(ref.Borrow)() }

func keep(value Pointer, destination *api.Worker) {
	_ = any(value)
	_ = Holder{Value: value}
	_ = api.Worker{Next: value}
	destination.Next = value
	_ = [1]Pointer{value}
	_ = []Pointer{value}
	_ = map[Pointer]int{value: 1}
	channel <- value
	_ = api.Box[Pointer]{Value: value}
	_ = api.Keep(value)
	_ = func() int { return value.Count() }
	go value.Count()
	defer value.Count()
	_ = &value
	_ = (*api.Alias)(value)
	_ = (*api.Worker)(nil)
}

func recoverPointer(value any) {
	_ = value.(Pointer)
	switch value.(type) {
	case Pointer:
	}
}

func indirect(value **api.Worker) { _ = value }

func variadic(values ...Pointer) { _ = values }

func start(group *supervise.Group, value Pointer) {
	_, _ = supervise.Start[api.PointerArgs](group, value)
}

func callback(read func(Pointer) int, value Pointer) { _ = read(value) }

// Passing a scalar-only closure cannot conceal a constant operand that captures the pointer.
func constant(value Pointer) { _ = func() int { return len([1]Pointer{value}) } }
