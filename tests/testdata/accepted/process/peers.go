package process

import (
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
	config := proc.Start[ConfigLoader](root)
	cache := proc.Start[Cache](proc.StartSupervisor(root))
	cache.Proc().SetConfig(config)
	return Peers{Config: config, Cache: cache, Limiter: proc.Start[Limiter](root)}
}

// Serve handles one request with direct calls on Proc results.
func Serve(peers Peers) (int, bool) {
	if peers.Limiter.Proc().Take(peers.Config.Proc().Limit("requests")) == Limited {
		return 0, false
	}
	usage := peers.Limiter.Proc().Usage(10)
	entry := peers.Cache.Proc().Latest()
	return entry.Version + usage.Used, true
}
