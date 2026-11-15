// Package trading runs a local market-making simulation with replaceable strategies.
package trading

import (
	"sync"

	"github.com/lucasavila00/immutago/pkg/im"
)

// Tick is one ordered market update. Trade is zero when no trade occurred.
// Prices and cash use integer cents; each simulated order contains one unit.
type Tick struct {
	im.Copied
	Sequence uint64 `json:"sequence"`
	Bid      int64  `json:"bid"`
	Ask      int64  `json:"ask"`
	Trade    int64  `json:"trade"`
}

// Parameters controls the quote distance and the adjustment for held inventory.
type Parameters struct {
	im.Copied
	Edge int64 `json:"edge"`
	Skew int64 `json:"skew"`
}

// Limits bounds inventory per instrument and the market spread used for quoting.
type Limits struct {
	im.Copied
	MaxInventory int64 `json:"max_inventory"`
	MaxSpread    int64 `json:"max_spread"`
}

// Quote contains one-unit orders. A zero price means that side has no order.
type Quote struct {
	im.Copied
	Sequence uint64 `json:"sequence"`
	Bid      int64  `json:"bid"`
	Ask      int64  `json:"ask"`
}

// InstrumentTotals contains independent numeric state for one instrument.
type InstrumentTotals struct {
	im.Copied
	Events    uint64 `json:"events"`
	Buys      uint64 `json:"buys"`
	Sells     uint64 `json:"sells"`
	Cancels   uint64 `json:"cancels"`
	Inventory int64  `json:"inventory"`
	Mid       int64  `json:"mid"`
	Quote     Quote  `json:"quote"`
}

// Totals reports one paper account, including cash reserved for all resting bids.
type Totals struct {
	im.Copied
	Budget    int64  `json:"budget"`
	Cash      int64  `json:"cash"`
	Reserved  int64  `json:"reserved"`
	Equity    int64  `json:"equity"`
	Inventory int64  `json:"inventory"`
	Events    uint64 `json:"events"`
	Buys      uint64 `json:"buys"`
	Sells     uint64 `json:"sells"`
	Cancels   uint64 `json:"cancels"`
}

// account survives worker retries. Its one lock owns cash and every instrument's orders.
type account struct {
	mu       sync.Mutex
	budget   int64
	cash     int64
	reserved int64
	markets  []InstrumentTotals
}

// newAccount allocates a separate paper budget and fixed instrument slots for one trial.
func newAccount(budget int64, instruments int) *account {
	return &account{budget: budget, cash: budget, markets: make([]InstrumentTotals, instruments)}
}

// step applies a new event once, fills prior quotes, and computes replacement quotes.
// Input validation and report allocation belong outside this calculation.
func (a *account) step(index int, tick Tick, parameters Parameters, limits Limits) Quote {
	a.mu.Lock()
	defer a.mu.Unlock()
	market := &a.markets[index]
	if tick.Sequence <= market.Quote.Sequence {
		return market.Quote
	}
	previous := market.Quote
	if tick.Trade != 0 {
		switch {
		case previous.Bid != 0 && tick.Trade <= previous.Bid:
			a.cash -= previous.Bid
			market.Inventory++
			market.Buys++
		case previous.Ask != 0 && tick.Trade >= previous.Ask:
			a.cash += previous.Ask
			market.Inventory--
			market.Sells++
		}
	}
	a.reserved -= previous.Bid
	market.Events++
	market.Mid = (tick.Bid + tick.Ask) / 2
	market.Quote = Quote{Sequence: tick.Sequence}
	if tick.Ask-tick.Bid > limits.MaxSpread {
		return market.Quote
	}
	fair := market.Mid - market.Inventory*parameters.Skew
	bid := min(tick.Bid, fair-parameters.Edge)
	ask := max(tick.Ask, fair+parameters.Edge)
	if market.Inventory < limits.MaxInventory && bid > 0 && bid <= a.cash-a.reserved {
		market.Quote.Bid = bid
		a.reserved += bid
	}
	if market.Inventory > 0 {
		market.Quote.Ask = ask
	}
	return market.Quote
}

// cancel removes a worker's resting orders while preserving its sequence and holdings.
func (a *account) cancel(index int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	market := &a.markets[index]
	if market.Quote.Bid != 0 || market.Quote.Ask != 0 {
		market.Cancels++
	}
	a.reserved -= market.Quote.Bid
	market.Quote.Bid, market.Quote.Ask = 0, 0
}

// read returns one complete account result without sharing mutable state.
func (a *account) read() Totals {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.totals()
}

// totals reads fields under the caller's account lock.
func (a *account) totals() Totals {
	result := Totals{Budget: a.budget, Cash: a.cash, Reserved: a.reserved, Equity: a.cash}
	for _, market := range a.markets {
		result.Equity += market.Inventory * market.Mid
		result.Inventory += market.Inventory
		result.Events += market.Events
		result.Buys += market.Buys
		result.Sells += market.Sells
		result.Cancels += market.Cancels
	}
	return result
}
