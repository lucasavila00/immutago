package use

import frozenstring "example.com/consumer/frozen_string"

// Read receives a frozen string without importing im directly.
func Read() string {
	text := frozenstring.TextValue()
	return text.Clone()
}
