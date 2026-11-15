package dependency

import (
	"example.com/consumer/pointer"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

var ref supervise.Ref[pointer.Worker]
