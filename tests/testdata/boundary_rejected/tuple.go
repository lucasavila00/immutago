package loader

import (
	"example.com/consumer/logx"
	"example.com/consumer/model"
)

func pair() (*model.Snapshot, int) { return &model.Snapshot{}, 1 }
func Leak()                        { logx.Values(pair()) }
