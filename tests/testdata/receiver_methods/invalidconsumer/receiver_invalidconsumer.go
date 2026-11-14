package invalidconsumer

import "example.com/consumer/invalid"

func Change() {
	product := &invalid.Product{}
	product.Change()
}
