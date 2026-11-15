package trading

import "testing"

// TestAccountReservesAcrossInstruments prevents two workers from spending the same cash.
func TestAccountReservesAcrossInstruments(t *testing.T) {
	account := newAccount(100, 2)
	parameters := Parameters{Edge: 1}
	limits := Limits{MaxInventory: 2, MaxSpread: 10}
	first := account.step(0, Tick{Sequence: 1, Bid: 99, Ask: 101}, parameters, limits)
	second := account.step(1, Tick{Sequence: 1, Bid: 99, Ask: 101}, parameters, limits)
	if first.Bid != 99 || second.Bid != 0 || account.read().Reserved != 99 {
		t.Fatal(first, second, account.read())
	}
	account.step(0, Tick{Sequence: 2, Bid: 99, Ask: 101, Trade: 99}, parameters, limits)
	if got := account.read(); got.Cash != 1 || got.Inventory != 1 || got.Buys != 1 || got.Reserved != 0 {
		t.Fatal(got)
	}
	account.step(0, Tick{Sequence: 3, Bid: 99, Ask: 101, Trade: 101}, parameters, limits)
	if got := account.read(); got.Cash != 102 || got.Inventory != 0 || got.Sells != 1 {
		t.Fatal(got)
	}
}

// TestReplayAndCancel preserve applied fills and sequence state across a worker retry.
func TestReplayAndCancel(t *testing.T) {
	account := newAccount(1000, 1)
	parameters := Parameters{Edge: 1}
	limits := Limits{MaxInventory: 2, MaxSpread: 10}
	tick := Tick{Sequence: 1, Bid: 99, Ask: 101}
	account.step(0, tick, parameters, limits)
	tick.Sequence, tick.Trade = 2, 99
	account.step(0, tick, parameters, limits)
	account.cancel(0)
	before := account.read()
	quote := account.step(0, tick, parameters, limits)
	if got := account.read(); got != before || quote.Bid != 0 || quote.Ask != 0 {
		t.Fatal("replay changed the account or recreated a canceled order", got, quote)
	}
	tick.Sequence = 3
	account.step(0, tick, parameters, limits)
	if got := account.read(); got.Buys != 1 || got.Inventory != 1 || got.Cash != 901 {
		t.Fatal("the canceled order filled", got)
	}
}
