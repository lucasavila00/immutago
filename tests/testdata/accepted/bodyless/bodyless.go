//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package bodyless

import (
	"context"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Config struct {
	im.Frozen
	Count int
}

type Plain struct{ Count int }
type Count = uint64

func cpuID() Count
func checksum([]byte) uint64
func update(*Plain, map[string]int) error
func (*Plain) Read() int

func ReadID() uint64 { return cpuID() }

// Worker has protected internal data that a Ref does not expose as a field.
type Worker struct {
	supervise.Service[supervise.NoArgs]
	config im.Cell[Config]
}

func (*Worker) Run(context.Context) {}

type Handle = supervise.Ref[Worker]
type Holder struct{ ref Handle }

func nativeReference(Handle) Handle
func nativeCallback(func() Handle)
func (*Holder) NativeRead() Handle
