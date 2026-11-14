package noref_consumer

import "example.com/consumer/invalid_noref"

func Read() int { return invalid_noref.Number() }
