package wrapped_noref_consumer

import "example.com/consumer/noref_wrapper"

func Read() int { return noref_wrapper.Number() }
