package freeze_wrapper

import "example.com/consumer/invalid_freeze"

type Projection = invalid_freeze.Projection

func Value() Projection { return invalid_freeze.Value() }
