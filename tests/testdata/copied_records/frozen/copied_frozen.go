//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package frozen

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Snapshot struct {
	im.Frozen
	Point   api.Point
	Points  [2]api.Point
	Score   api.Score
	Summary api.Summary
}
