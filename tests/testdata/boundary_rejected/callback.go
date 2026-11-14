package loader

import (
	"example.com/consumer/logx"
	"example.com/consumer/model"
)

func makeValue() *model.Shared { return &model.Shared{} }
func Leak()                    { logx.Info("callback", makeValue) }
