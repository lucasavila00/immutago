package borrow_consumer

import "example.com/consumer/invalid_borrow"

func Read() int { return invalid_borrow.Number() }
