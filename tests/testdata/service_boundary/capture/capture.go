package capture

import "example.com/consumer/api"

// saved retains a service pointer obtained through an imported Ref.
var saved = api.Get().Borrow()
