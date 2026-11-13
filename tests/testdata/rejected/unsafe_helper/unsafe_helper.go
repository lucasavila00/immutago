package unsafe_helper

import "unsafe" // want "unsupported unsafe import"

func Pointer() unsafe.Pointer { return nil }
