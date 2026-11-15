package loader

import (
	"example.com/consumer/logx"
	"example.com/consumer/model"
)

func makeValue() model.Local { return model.Local{} }
func Leak()                  { logx.Info("callback", makeValue) }
