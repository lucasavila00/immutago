package shape_consumer

import "example.com/consumer/invalid_shape"

func Read() int { return invalid_shape.Number() }
