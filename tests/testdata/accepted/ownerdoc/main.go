package main

import "github.com/lucasavila00/immutago/pkg/im"

type Config struct {
	im.Frozen
	Ready bool
	Limit int
}

var current im.Cell[Config]

func allowed(used int) bool {
	config := current.Load()
	ready := im.Copy(im.Select(config, func(value Config) bool { return value.Ready }))
	if !ready {
		return false
	}
	limit := im.Copy(im.Select(config, func(value Config) int { return value.Limit }))
	return used < limit
}

func main() {
	current.Store(Config{Ready: true, Limit: 100})
	println(allowed(80)) // true
	current.Store(Config{Ready: true, Limit: 50})
	println(allowed(80)) // false
}
