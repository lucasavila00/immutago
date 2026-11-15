package main

import "github.com/lucasavila00/immutago/pkg/im"

type Config struct {
	im.Scoped
	Limit int
}

var current im.Owner[Config]

func allowed(used int) bool {
	config := current.Load()
	if config == nil {
		return false
	}
	return used < config.Limit
}

func main() {
	current.Store(&Config{Limit: 100})
	println(allowed(80)) // true
	current.Store(&Config{Limit: 50})
	println(allowed(80)) // false
}
