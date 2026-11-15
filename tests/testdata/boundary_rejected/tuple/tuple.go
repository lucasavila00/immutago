package loader

import (
	"example.com/consumer/logx"
	"example.com/consumer/model"
)

func pair() (model.Local, int) { return model.Local{}, 1 }
func Leak()                    { logx.Values(pair()) }
