package randomunsafe

import "unsafe"

func Size() uintptr { return unsafe.Sizeof(0) }
