package servicecross

import (
	"time"

	"example.com/consumer/service"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Version reads a service from another package through its Ref.
func Version(root *supervise.Group) int {
	config, err := supervise.Start[service.ConfigLoader](root, service.LoaderArgs{Path: "requests", Every: time.Second})
	if err != nil {
		panic(err)
	}
	return config.Current().Version()
}
