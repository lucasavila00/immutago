package freeze_wrapper

import "example.com/consumer/invalid_freeze"

type View = invalid_freeze.View

func Value() View { return invalid_freeze.Value() }
