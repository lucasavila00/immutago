package valid

import "example.com/consumer/api"

// Dependency passes an imported dependency alias without importing proc directly.
func Dependency() api.Dependency {
	value := api.Wrap(new(int))
	return api.Copy(value)
}
