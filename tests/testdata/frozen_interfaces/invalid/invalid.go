//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package invalid

import (
	"example.com/consumer/api"
)

type Hidden struct{ api.Scorer }

func HiddenValue(s api.Scorer) api.Scorer { return Hidden{Scorer: s} }
