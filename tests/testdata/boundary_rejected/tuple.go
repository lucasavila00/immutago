package loader

import (
	"example.com/consumer/logx"
	"example.com/consumer/model"
)

func pair() (*model.Frozen, int) { return &model.Frozen{}, 1 }
func Leak()                      { logx.Values(pair()) }
