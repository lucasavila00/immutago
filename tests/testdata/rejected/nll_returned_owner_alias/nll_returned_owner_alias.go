package rejected

import "example.com/consumer/borrow_api"

func bad() {
	owner := borrow_api.Fresh(1)
	shared := owner
	owner.X = 2 // want "borrow|shared reference"
	_ = borrow_api.Read(shared)
}
