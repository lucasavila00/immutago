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

func BranchLastUse(condition bool) Good {
	var value Good
	pointer := &value
	if condition {
		_ = read(pointer)
	} else {
		_ = read(pointer)
	}
	value.X = 1
	return value
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

func LoopLastUse(count int) Good {
	var value Good
	pointer := &value
	for index := 0; index < count; index++ {
		_ = read(pointer)
	}
	value.X = count
	return value
}
