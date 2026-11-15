package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func opaque(value im.FrozenSlice[int])

type Config struct {
	im.Snapshot
	Count int
}

type Alias = *Config

func read(Alias) int
func makeConfig() Config
func (*Config) CountValue() int
