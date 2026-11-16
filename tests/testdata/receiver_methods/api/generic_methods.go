package api

func (p Product) Map[T any](read func(int) T) T         { return read(p.Price) }
func (p *Product) MapPointer[T any](read func(int) T) T { return read(p.Price) }
