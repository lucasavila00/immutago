package rejected

func (p Product) Map[T any](read func(int) T) T          { return read(p.Price) }
func (p *Product) MapPointer[T any](read func(int) T) T  { return read(p.Price) }
func (s Settings) Map[T any](read func(int) T) T         { return read(s.Limit) }
func (s *Settings) MapPointer[T any](read func(int) T) T { return read(s.Limit) }
func (p Product) ReadLocal[T any](read func(Settings) T) T {
	return read(Settings{Limit: p.Price})
}
func (p Product) KeepLocal[T Settings](value T) T { return value }
func (p Product) ChangeGeneric[T any](value T)    { p.Price++ }
func (p *Product) ChangePointer[T any](value T)   { p.Price++ }
func (p Product) AddressGeneric[T any](value T) *Product {
	return &p
}

func local(n int) Settings { return Settings{Limit: n} }
func number(n int) int     { return n }

func genericCalls(value Product, settings *Settings) {
	_ = value.Map[Settings](local)
	_ = settings.MapPointer[int](number)
	local := Settings{Limit: 1}
	_ = local.Map[int](number)
	_ = value.MapPointer[int](number)
}

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen
