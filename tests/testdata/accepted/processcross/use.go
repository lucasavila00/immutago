package processcross

import (
	"time"

	"example.com/consumer/process"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// Version reads a process from another package through its PID.
func Version(root *proc.Supervisor) int {
	config := proc.Start[process.ConfigLoader](root, process.LoaderArgs{Path: "requests", Every: time.Second})
	return config.Proc().Version()
}
