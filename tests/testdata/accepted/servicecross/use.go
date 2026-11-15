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
	return counter.Current().Read().Count
}
