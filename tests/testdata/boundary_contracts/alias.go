package contract

import (
	"example.com/consumer/model"
)

var _ model.Scalar[int]

func missing(model.Scalar[int])
