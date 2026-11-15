package loader

import (
	"example.com/consumer/logx"
	"example.com/consumer/model"
)

func makeValue() *model.Snapshot { return &model.Snapshot{} }
func Leak()                      { logx.Info("callback", makeValue) }
