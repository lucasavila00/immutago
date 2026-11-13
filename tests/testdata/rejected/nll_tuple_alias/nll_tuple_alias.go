package rejected

import "example.com/consumer/borrow_api"

func bad() {
	first, second := borrow_api.FreshPair()
	first.X = 2 // want "borrow|shared reference"
	_ = borrow_api.Read(second)
}
