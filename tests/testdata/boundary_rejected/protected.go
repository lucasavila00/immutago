package loader

import "example.com/consumer/logx"

func Leak(l *Loader) {
	current := l.Current.Load()
	logx.Info("model", current)
}
