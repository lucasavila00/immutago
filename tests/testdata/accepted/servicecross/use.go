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
	counter.Borrow().Update()
	return counter.Borrow().Read().Count
}

func route(pool supervise.Ref[service.Pool]) int {
	selected := pool.Borrow().Route("worker")
	return selected.Worker.Borrow().Count()
}
