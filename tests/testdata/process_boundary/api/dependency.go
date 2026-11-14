package api

import "github.com/lucasavila00/immutago/pkg/proc"

// Dependency keeps the runtime identity through an exported alias.
type Dependency = proc.UncheckedDependency[*int]

// Wrap returns a dependency without exposing its value.
func Wrap(value *int) Dependency { return proc.Trust(value) }

// Copy passes a dependency through an ordinary function.
func Copy(value Dependency) Dependency { return value }
