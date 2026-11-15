package loader

import (
	"example.com/consumer/logx"
	"example.com/consumer/model"
)

func makeValue() model.Frozen { return model.Frozen{} }
func Leak()                   { logx.Info("callback", makeValue) }
