package invalidconsumer

import "example.com/consumer/invalid"

func Read() float64 { return invalid.Make().Score(nil) }
