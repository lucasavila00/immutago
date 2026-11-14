package deep

import "github.com/lucasavila00/immutago/pkg/im"

// Note is a marked struct, so the checker checks this package.
type Note struct {
	im.Shared
	Text string
}

var counts = map[string]int{}

// Tracker counts keys in a package map.
type Tracker struct{}

// Add changes the package map.
func (Tracker) Add(key string) { counts[key]++ }
