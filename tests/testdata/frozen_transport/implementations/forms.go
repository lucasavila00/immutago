package implementations

import "example.com/consumer/api"

func take(api.Reader) {}
func direct(value api.Promoted) {
	var reader api.Reader = value
	_ = reader
	take(api.Identity(value))
}
func result(value api.Promoted) api.Reader { return api.Identity(value) }
func capture(value api.Promoted) func() api.Reader {
	return func() api.Reader { return value }
}
func asserted(value any) {
	_ = value.(api.Reader)
	reader, ok := value.(api.Reader)
	_, _ = reader, ok
}
func switched(value any) {
	switch value.(type) {
	case api.Reader:
	}
}

func generic[T api.Reader](value T) api.Reader { return value }
