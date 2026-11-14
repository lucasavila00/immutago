package valid

import "example.com/consumer/api"

// Read uses an imported PID without importing proc or im directly.
func Read() int { return api.Get().Proc().Count() }
