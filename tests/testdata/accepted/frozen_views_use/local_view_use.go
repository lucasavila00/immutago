package localviewuse

import "example.com/consumer/frozen_views"

// Read uses a local result without directly importing im.
func Read() string {
	value := localviews.TextValue("text")
	return value.Clone()
}

// Record passes an inferred record through another checked package.
func Record() string {
	value := localviews.Pass(localviews.Reply{})
	return value.Text.Clone()
}
