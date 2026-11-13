package rejected

import "example.com/consumer/borrow_api"

func bad() {
	var value borrow_api.Good
	pointer := borrow_api.Field(&value)
	value.X = 2 // want "borrow|shared reference"
	_ = *pointer
}
