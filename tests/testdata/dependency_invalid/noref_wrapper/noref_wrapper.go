package noref_wrapper

import "example.com/consumer/invalid_noref"

func Number() int { return invalid_noref.Number() }
