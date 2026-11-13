package generic_callback_api

type Getter interface{ Ptr() *int }

func Apply[T Getter](get T) { *get.Ptr() = 1 }
