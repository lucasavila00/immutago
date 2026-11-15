package valid

import "example.com/consumer/api"

// Read uses an imported Ref without importing supervise or im directly.
func Read() int { return api.Get().Current().Count() }

type Producer interface{ Worker() api.Handle }

func Produced(producer Producer) int {
	other := producer
	return other.Worker().Current().Count()
}
