package review_convergence

import "github.com/lucasavila00/immutago/pkg/im"

type ConversionValue struct {
	im.Shared
	N int
}

func retainArguments(...any)                 {}
func conversionPair() (int, ConversionValue) { return 1, ConversionValue{} }
func makeConversionValue() ConversionValue   { return ConversionValue{} }
func boxValue(value ConversionValue)         { retainArguments(value) }
func boxTuple()                              { retainArguments(conversionPair()) }
func boxCallback()                           { retainArguments(makeConversionValue) }
func switchInterface(value ConversionValue, boxed any) {
	switch boxed {
	case value:
	}
}
