package process

import (
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// Peers keeps the PIDs of the service processes.
type Peers struct {
	im.Shared
	Config  proc.PID[ConfigLoader]
	Cache   proc.PID[Cache]
	Limiter proc.PID[Limiter]
}

// StartPeers starts the processes under root.
func StartPeers(root *proc.Supervisor) Peers {
	config := proc.Start[ConfigLoader](root, LoaderArgs{Path: "requests", Every: time.Second})
	return Peers{
		Config:  config,
		Cache:   proc.Start[Cache](proc.StartSupervisor(root), proc.None{}),
		Limiter: proc.Start[Limiter](root, LimiterArgs{Loader: config, Key: "requests"}),
	}
}

// Serve handles one request with direct calls on Proc results.
func Serve(peers Peers) (int, bool) {
	if peers.Limiter.Proc().Take() == Limited {
		return 0, false
	}
	usage := peers.Limiter.Proc().Usage()
	entry := peers.Cache.Proc().Latest()
	return entry.Version + usage.Used, true
}
