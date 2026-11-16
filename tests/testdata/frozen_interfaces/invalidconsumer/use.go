package invalidconsumer

import "example.com/consumer/invalid"

func Read() float64 { return invalid.HiddenValue(nil).Score(nil) }
