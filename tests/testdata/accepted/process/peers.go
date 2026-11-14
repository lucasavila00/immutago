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
	config, err := proc.Start[ConfigLoader](root, LoaderArgs{Path: "requests", Every: time.Second})
	if err != nil {
		panic(err)
	}
	cache, err := proc.Start[Cache](proc.StartSupervisor(root), proc.None{})
	if err != nil {
		panic(err)
	}
	limiter, err := proc.Start[Limiter](root, LimiterArgs{Loader: config, Key: "requests"})
	if err != nil {
		panic(err)
	}
	return Peers{Config: config, Cache: cache, Limiter: limiter}
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
