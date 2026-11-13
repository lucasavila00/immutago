package borrowcases

func Branch(condition bool) *Good {
	value := &Good{}
	if condition {
		value = &Good{X: 1}
	}
	return value
}
func Loop(count int) Good {
	var value Good
	for index := 0; index < count; index++ {
		value = Good{X: index}
	}
	return value
}
func EarlyReturn(condition bool) *Good {
	value := &Good{}
	if condition {
		return value
	}
	return &Good{X: 2}
}
