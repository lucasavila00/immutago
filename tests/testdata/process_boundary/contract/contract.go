package contract

import "example.com/consumer/api"

var value api.Contract[int]

// missingBody is forbidden because the used alias contract includes a PID.
func missingBody()
