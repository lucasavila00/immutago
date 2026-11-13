package rejected

import `unsafe` // want "unsupported unsafe import"

var _ unsafe.Pointer
