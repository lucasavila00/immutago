package loader

import "example.com/consumer/logx"

func Leak(l *Loader) { logx.Info("loader", l) }
