package borrowcaller

import "example.com/consumer/borrowapi"

func CrossPackage() borrowapi.Good {
	value := borrowapi.Make(1)
	pointer := &value
	_ = borrowapi.Read(pointer)
	value.X = 2
	return value
}
