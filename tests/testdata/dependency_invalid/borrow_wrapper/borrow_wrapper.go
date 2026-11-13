package borrow_wrapper

import "example.com/consumer/invalid_borrow"

func Number() int { return invalid_borrow.Number() }
