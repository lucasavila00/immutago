package bodyless

import "github.com/lucasavila00/immutago/pkg/im"

type Config struct {
	im.Shared
	Count int
}

type Plain struct{ Count int }
type Count = uint64

func cpuID() Count
func checksum([]byte) uint64
func update(*Plain, map[string]int) error
func (*Plain) Read() int

func ReadID() uint64 { return cpuID() }
