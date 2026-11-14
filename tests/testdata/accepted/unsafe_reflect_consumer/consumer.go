package consumer

import "example.com/consumer/unsafe_reflect"

// Read accepts an immutable pointer from a package that also uses unsafe and reflect.
func Read(record *ordinary.Record) byte { return ordinary.NameFirst(record) }
