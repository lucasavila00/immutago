package servicecross

import (
	"example.com/consumer/service"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

func Count(root *supervise.Group) int {
	counter, err := supervise.Start[service.Counter](root, map[string]int{})
	if err != nil {
		panic(err)
	}
	counter.Current().Update()
	return counter.Current().Read().Count
}

func route(pool supervise.Ref[service.Pool]) int {
	selected := pool.Current().Route("worker")
	return selected.Worker.Current().Count()
}
