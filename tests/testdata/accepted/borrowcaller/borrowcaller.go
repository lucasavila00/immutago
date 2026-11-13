package borrowcaller

import "example.com/consumer/borrowapi"

func CrossPackage() borrowapi.Good {
	var value borrowapi.Good
	pointer := borrowapi.Identity(&value)
	_ = borrowapi.Read(pointer)
	value.X = 2
	generic := borrowapi.GenericIdentity(&value)
	_ = borrowapi.Read(generic)
	value.X = 3
	field := borrowapi.Field(&value)
	_ = *field
	value.X = 4
	_ = borrowapi.Read(borrowapi.Fresh(4))
	return value
}

func ReadFreshFields() int {
	first, second := borrowapi.FreshFields()
	return *first + *second
}

func ConstructReturnedOwner() *borrowapi.Good {
	owner := borrowapi.Fresh(1)
	owner.X = 2
	shared := owner
	_ = borrowapi.Read(shared)
	owner.X = 3
	return owner
}

func IndependentReturnedOwners() *borrowapi.Good {
	first, second := borrowapi.SeparateFresh()
	first.X = 2
	_ = borrowapi.Read(second)
	return first
}
