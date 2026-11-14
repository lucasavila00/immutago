package contract

import "example.com/consumer/api"

var value api.Contract[int]

// missingBody has a Ref in its argument alias constraint.
func missingBody(value api.Contract[int])
