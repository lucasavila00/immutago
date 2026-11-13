package wrapped_borrow_consumer

import "example.com/consumer/borrow_wrapper"

func Read() int { return borrow_wrapper.Number() }
