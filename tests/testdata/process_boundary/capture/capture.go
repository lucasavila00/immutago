package capture

import "example.com/consumer/api"

// saved retains a process pointer obtained through an imported PID.
var saved = api.Get().Proc()
