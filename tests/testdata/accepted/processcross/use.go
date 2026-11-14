package processcross

import (
	"time"

	"example.com/consumer/process"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// Version reads a process from another package through its PID.
func Version(root *proc.Supervisor) int {
	config, err := proc.Start[process.ConfigLoader](root, process.LoaderArgs{Path: "requests", Every: time.Second})
	if err != nil {
		panic(err)
	}
	return config.Proc().Version()
}
