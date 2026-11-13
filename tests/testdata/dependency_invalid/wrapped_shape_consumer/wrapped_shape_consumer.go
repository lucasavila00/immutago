package wrapped_shape_consumer

import "example.com/consumer/shape_wrapper"

func Read() int { return shape_wrapper.Number() }
