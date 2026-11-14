package service

import (
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Peers keeps the Refs of the service services.
type Peers struct {
	im.Shared
	Config  supervise.Ref[ConfigLoader]
	Cache   supervise.Ref[Cache]
	Limiter supervise.Ref[Limiter]
}

// StartPeers starts the services under root.
func StartPeers(root *supervise.Group) Peers {
	config, err := supervise.Start[ConfigLoader](root, LoaderArgs{Path: "requests", Every: time.Second})
	if err != nil {
		panic(err)
	}
	cache, err := supervise.Start[Cache](supervise.StartGroup(root), supervise.NoArgs{})
	if err != nil {
		panic(err)
	}
	limiter, err := supervise.Start[Limiter](root, LimiterArgs{Loader: config, Key: "requests"})
	if err != nil {
		panic(err)
	}
	return Peers{Config: config, Cache: cache, Limiter: limiter}
}

// Serve handles one request with direct calls on Current results.
func Serve(peers Peers) (int, bool) {
	if peers.Limiter.Current().Take() == Limited {
		return 0, false
	}
	usage := peers.Limiter.Current().Usage()
	entry := peers.Cache.Current().Latest()
	return entry.Version + usage.Used, true
}
