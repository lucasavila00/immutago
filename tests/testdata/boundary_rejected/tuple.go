package loader

import (
	"example.com/consumer/logx"
	"example.com/consumer/model"
)

func pair() (*model.Shared, int) { return &model.Shared{}, 1 }
func Leak()                      { logx.Values(pair()) }
