package borrowcaller

import "example.com/consumer/borrowapi"

func CrossPackage() borrowapi.Good {
	value := borrowapi.Make(1)
	pointer := &borrowapi.Good{X: value.X}
	_ = borrowapi.Read(pointer)
	value = borrowapi.Good{X: 2}
	return value
}

func CrossPackagePointer() *borrowapi.Good {
	value := borrowapi.Shared(1)
	return borrowapi.Forward(value)
}
