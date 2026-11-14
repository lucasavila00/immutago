package processcross

import (
	"example.com/consumer/process"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// Version reads a process from another package through its PID.
func Version(root *proc.Supervisor) int {
	config := proc.Start[process.ConfigLoader](root)
	return config.Proc().Version()
}
