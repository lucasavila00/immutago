package unsafe_consumer

import "example.com/consumer/invalid_unsafe"

func Read() int { return invalid_unsafe.Number() }
