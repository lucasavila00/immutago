//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func opaque(value im.FrozenSlice[int])

type Config struct {
	im.Frozen
	Count int
}

type Alias = *Config

func read(Alias) int
func makeConfig() Config
func (*Config) CountValue() int
