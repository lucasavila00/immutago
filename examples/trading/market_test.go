package trading

import (
	"sync/atomic"
	"testing"
)

// testAccount creates funded state for direct risk and fill tests.
func testAccount(cash, budget int64, symbols ...string) *account {
	spec := strategySpec{Budget: budget}
	for _, symbol := range symbols {
		spec.Markets = append(spec.Markets, marketSpec{Symbol: symbol})
	}
	return newAccount(spec, Balance{Cash: cash}, 42, new(atomic.Uint64))
}

// TestAccountReservesAcrossInstruments prevents two workers from spending the same cash.
func TestAccountReservesAcrossInstruments(t *testing.T) {
	account := testAccount(100, 1000, "AAA", "BBB")
	limits := Limits{MaxInventory: 2, MaxSpread: 10}
	first, ok := account.reserve(0, Buy, 99, limits)
	if !ok {
		t.Fatal("first order rejected")
	}
	if second, ok := account.reserve(1, Buy, 99, limits); ok || second.ID.Sequence != 0 || account.read().Reserved != 99 {
		t.Fatal(second, ok, account.read())
	}
	account.apply(0, OrderState{ID: first.ID, Filled: 1, Value: 99})
	if got := account.read(); got.Cash != 1 || got.Inventory != 1 || got.Buys != 1 || got.Reserved != 0 {
		t.Fatal(got)
	}
	sell, ok := account.reserve(0, Sell, 101, limits)
	if !ok {
		t.Fatal("sell order rejected")
	}
	account.apply(0, OrderState{ID: sell.ID, Filled: 1, Value: 101})
	if got := account.read(); got.Cash != 102 || got.Inventory != 0 || got.Sells != 1 {
		t.Fatal(got)
	}
}

// TestCumulativeFillsIgnoreDuplicates includes a late old fill after its slot has been reused.
func TestCumulativeFillsIgnoreDuplicates(t *testing.T) {
	account := testAccount(1000, 1000, "AAA")
	limits := Limits{MaxInventory: 5, MaxSpread: 10}
	order, _ := account.reserve(0, Buy, 99, limits)
	account.apply(0, OrderState{ID: order.ID, Open: true})
	fill := OrderState{ID: order.ID, Filled: 1, Value: 99}
	account.apply(0, fill)
	next, _ := account.reserve(0, Buy, 99, limits)
	before := account.read()
	account.apply(0, fill)
	account.apply(0, OrderState{ID: order.ID, Open: true})
	if got := account.read(); got != before || next.ID == order.ID {
		t.Fatal(got, before, next)
	}
}

// TestBudgetAndInventoryLimits checks the risk boundaries without a venue transport.
func TestBudgetAndInventoryLimits(t *testing.T) {
	account := testAccount(1000, 100, "AAA", "BBB")
	limits := Limits{MaxInventory: 1, MaxSpread: 10}
	order, _ := account.reserve(0, Buy, 99, limits)
	if _, ok := account.reserve(1, Buy, 99, limits); ok {
		t.Fatal("quote budget was spent twice")
	}
	account.apply(0, OrderState{ID: order.ID, Filled: 1, Value: 99})
	if _, ok := account.reserve(0, Buy, 99, limits); ok {
		t.Fatal("inventory limit was exceeded")
	}
}
