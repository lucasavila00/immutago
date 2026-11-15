package api

// Dependency keeps the pointer type through an exported alias.
type Dependency = *int

// Wrap returns a dependency with the same pointer.
func Wrap(value *int) Dependency { return value }

// Copy passes a dependency through an ordinary function.
func Copy(value Dependency) Dependency { return value }
