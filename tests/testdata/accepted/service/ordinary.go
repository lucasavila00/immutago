package service

// Services and dependencies use ordinary Go storage and transport.
var saved *Counter
var dependency map[string]int

func construct() *Counter            { return &Counter{} }
func store(c *Counter)               { saved = c; dependency = map[string]int{} }
func readDependency() map[string]int { return dependency }
