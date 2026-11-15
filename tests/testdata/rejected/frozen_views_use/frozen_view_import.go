package viewuse

import "example.com/consumer/frozen_views/api"

// Inferred types carry the local rule even when this package does not import im.
var text = viewapi.Text()
var record = viewapi.Result()

// Save cannot assign an imported result to a package variable.
func Save() { text = viewapi.Text() }
