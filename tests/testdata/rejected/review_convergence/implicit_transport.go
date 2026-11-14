package review_convergence

func pointerInterfaceEqual(pointer *Value, boxed any) bool {
	return pointer == boxed
}

func valueInterfaceNotEqual(value Value, boxed any) bool {
	return boxed != value
}

func protectedPair() (Value, int) { return Value{X: 1}, 2 }

func forwardToCallback(callback func(Value, int)) {
	callback(protectedPair())
}
