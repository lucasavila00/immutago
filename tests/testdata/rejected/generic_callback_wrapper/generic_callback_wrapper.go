package generic_callback_wrapper

import "example.com/consumer/generic_callback_api"

func Apply[T generic_callback_api.Getter](get T) { generic_callback_api.Apply(get) }
