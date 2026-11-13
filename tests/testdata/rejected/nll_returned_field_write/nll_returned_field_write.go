package rejected

import "example.com/consumer/borrow_api"

func bad() {
	first, second := borrow_api.FreshFields()
	*first = 2 // want "borrow|shared reference"
	_ = *second
}
