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

func LoopAliasLastUse(count int) Good {
	var value Good
	var previous *Good
	for index := 0; index < count; index++ {
		if previous != nil {
			_ = read(previous)
		}
		value.X = index
		pointer := &value
		previous = pointer
	}
	return value
}
