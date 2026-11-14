package plain

import "example.com/consumer/state"

// Names is a mutable map of a package that the checker does not check.
var Names = map[string]string{}

// Version is an exported string of a package that the checker does not check.
var Version = "1"

// Remember changes Names.
func Remember(key string) { Names[key] = key }

// Forward calls a function of a checked package that changes a package map.
func Forward(key string) { state.Record(key) }
