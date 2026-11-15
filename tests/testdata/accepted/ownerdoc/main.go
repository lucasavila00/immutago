//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package main

import "github.com/lucasavila00/immutago/pkg/im"

type Config struct {
	im.Frozen
	Ready bool
	Limit int
}

var current ConfigCell

func allowed(used int) bool {
	config := current.Load()
	ready := config.Ready()
	if !ready {
		return false
	}
	limit := config.Limit()
	return used < limit
}

func main() {
	current.Store(Config{Ready: true, Limit: 100})
	println(allowed(80)) // true
	current.Store(Config{Ready: true, Limit: 50})
	println(allowed(80)) // false
}
