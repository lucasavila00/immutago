package review_convergence

func acceptCallback(func(ConversionValue)) {}
func passNilCallback()                     { acceptCallback(nil) }
func returnCallback() (callback func(ConversionValue)) {
	return
}

type CallbackHolder struct {
	Callback func(ConversionValue)
}

var callbackHolder CallbackHolder
var callbackChannel chan func(ConversionValue)
var callbackPointer *func(ConversionValue)

func returnCallbackHolder() (holder CallbackHolder) {
	return
}
