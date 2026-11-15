package service

import "github.com/lucasavila00/immutago/pkg/supervise"

// Services and dependency wrappers use ordinary Go storage and transport.
var saved *Counter
var dependency supervise.ConcurrentSafe[map[string]int]

func construct() *Counter            { return &Counter{} }
func store(c *Counter)               { saved = c; dependency = supervise.AssumeConcurrentSafe(map[string]int{}) }
func readDependency() map[string]int { return dependency.Value() }
