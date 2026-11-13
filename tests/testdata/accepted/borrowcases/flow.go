package borrowcases

func BranchLocal(condition bool) Good {
	var value Good
	if condition {
		pointer := &value
		_ = read(pointer)
	} else {
		value.X = 1
	}
	value.X = 2
	return value
}

func BranchMerge(condition bool) Good {
	var first, second Good
	var pointer *Good
	if condition {
		pointer = &first
	} else {
		pointer = &second
	}
	_ = read(pointer)
	first.X = 1
	second.X = 2
	return first
}

func LoopLocal(count int) Good {
	var value Good
	for index := 0; index < count; index++ {
		value.X = index
		pointer := &value
		_ = read(pointer)
	}
	value.X = count
	return value
}

func EarlyReturn(condition bool) Good {
	var value Good
	pointer := &value
	if condition {
		_ = read(pointer)
		return value
	}
	value.X = 2
	return value
}

func ShadowCopy() Good {
	value := Good{X: 1}
	pointer := &value
	{
		value := value
		value.X = 2
		_ = value
	}
	_ = read(pointer)
	return value
}

func BoundReadMethod() Good {
	var value Good
	method := value.Read
	_ = method()
	value.X = 2
	return value
}

func ClosureLastUse() Good {
	var value Good
	pointer := &value
	readLater := func() int { return pointer.X }
	_ = readLater()
	value.X = 2
	return value
}

func InterfaceLastUse() Good {
	var value Good
	var boxed any = &value
	_ = read(boxed.(*Good))
	value.X = 2
	return value
}

func ScalarInterfaceLastUse() Good {
	var value Good
	var boxed any = &value.X
	_ = readInt(boxed.(*int))
	value.X = 2
	return value
}
