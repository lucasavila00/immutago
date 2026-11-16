// Package trading runs a market maker bot with replaceable strategy workers.
package trading

import (
	"sync"
	"sync/atomic"

	"github.com/lucasavila00/immutago/pkg/im"
)

// Parameters controls quote distance and the adjustment for held inventory.
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

// Quote reports current orders. A zero price means that side has no order.
type Quote struct {
	im.Copied
	Sequence      uint64  `json:"sequence"`
	PolicyVersion uint64  `json:"policy_version"`
	Bid           int64   `json:"bid"`
	Ask           int64   `json:"ask"`
	BidID         OrderID `json:"bid_id"`
	AskID         OrderID `json:"ask_id"`
	BidPending    bool    `json:"bid_pending"`
	AskPending    bool    `json:"ask_pending"`
}

// InstrumentTotals contains independent numeric state for one instrument.
type InstrumentTotals struct {
	im.Copied
	Events    uint64 `json:"events"`
	Orders    uint64 `json:"orders"`
	Buys      int64  `json:"buys"`
	Sells     int64  `json:"sells"`
	Cancels   uint64 `json:"cancels"`
	Inventory int64  `json:"inventory"`
	Mid       int64  `json:"mid"`
	Quote     Quote  `json:"quote"`
}

// Totals reports actual account cash and cash reserved for pending or resting bids.
type Totals struct {
	im.Copied
	MaxReservedCash int64  `json:"max_reserved_cash"`
	Cash            int64  `json:"cash"`
	Reserved        int64  `json:"reserved"`
	Inventory       int64  `json:"inventory"`
	Events          uint64 `json:"events"`
	Orders          uint64 `json:"orders"`
	Buys            int64  `json:"buys"`
	Sells           int64  `json:"sells"`
	Cancels         uint64 `json:"cancels"`
}

// orderSlot retains an uncertain placement until the venue confirms its outcome.
type orderSlot struct {
	order    Order
	filled   int64
	value    int64
	reserved int64
	pending  bool
}

// instrument retains holdings and at most one pending or resting order per side.
type instrument struct {
	totals  InstrumentTotals
	last    Tick
	buy     orderSlot
	sell    orderSlot
	problem error
}

// account keeps cash and all instrument orders under one shared lock.
// Calls to the venue run outside this lock while their reservations remain held.
type account struct {
	mu              sync.Mutex
	maxReservedCash int64
	cash            int64
	reserved        int64
	session         uint64
	sequence        *atomic.Uint64
	markets         []instrument
}

// newAccount copies venue balances into the selected account's persistent state.
func newAccount(spec strategySpec, balance Balance, session uint64, sequence *atomic.Uint64) *account {
	result := &account{maxReservedCash: spec.MaxReservedCash, cash: balance.Cash, session: session, sequence: sequence,
		markets: make([]instrument, len(spec.Markets))}
	for index, market := range spec.Markets {
		result.markets[index].totals.Inventory = balance.Positions[market.Symbol]
	}
	return result
}

// plan records a new market update and computes quotes from one complete risk policy.
func (a *account) plan(index int, tick Tick, parameters Parameters, limits Limits, version uint64) (Quote, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	market := &a.markets[index]
	if tick.Sequence <= market.totals.Quote.Sequence {
		return a.quote(index), false
	}
	market.last = tick
	market.totals.Events++
	market.totals.Mid = (tick.Bid + tick.Ask) / 2
	market.totals.Quote.Sequence, market.totals.Quote.PolicyVersion = tick.Sequence, version
	return a.desired(index, parameters, limits), true
}

// desired computes one quote pair under the caller's account lock.
func (a *account) desired(index int, parameters Parameters, limits Limits) Quote {
	market := &a.markets[index]
	tick := market.last
	result := Quote{Sequence: tick.Sequence, PolicyVersion: market.totals.Quote.PolicyVersion}
	if tick.Ask-tick.Bid > limits.MaxSpread {
		return result
	}
	fair := market.totals.Mid - market.totals.Inventory*parameters.Skew
	bid := min(tick.Bid, fair-parameters.Edge)
	if market.totals.Inventory < limits.MaxInventory && bid > 0 &&
		bid <= min(a.cash, a.maxReservedCash)-a.reserved+market.buy.reserved {
		result.Bid = bid
	}
	if market.totals.Inventory > 0 {
		result.Ask = max(tick.Ask, fair+parameters.Edge)
	}
	return result
}

// replan includes fills confirmed while canceling the previous quote pair.
func (a *account) replan(index int, parameters Parameters, limits Limits) Quote {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.desired(index, parameters, limits)
}

// reserve checks shared cash again after earlier cancellation calls have completed.
func (a *account) reserve(index int, side Side, price int64, limits Limits) (Order, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	market := &a.markets[index]
	slot := &market.buy
	if side == Sell {
		slot = &market.sell
	}
	if price == 0 || slot.order.ID.Sequence != 0 {
		return Order{}, false
	}
	if side == Buy && (market.totals.Inventory >= limits.MaxInventory ||
		price > min(a.cash, a.maxReservedCash)-a.reserved) {
		return Order{}, false
	}
	if side == Sell && market.totals.Inventory <= 0 {
		return Order{}, false
	}
	sequence := a.sequence.Add(1)
	order := Order{ID: OrderID{Session: a.session, Sequence: sequence}, Side: side, Price: price, Quantity: 1}
	*slot = orderSlot{order: order, pending: true}
	if side == Buy {
		slot.reserved = price
		a.reserved += price
	}
	market.totals.Orders++
	return order, true
}

// apply uses cumulative fill deltas so acknowledgments and duplicate reports agree.
func (a *account) apply(index int, state OrderState) {
	a.mu.Lock()
	defer a.mu.Unlock()
	market := &a.markets[index]
	slot := &market.buy
	side := Buy
	if slot.order.ID != state.ID {
		slot, side = &market.sell, Sell
	}
	if slot.order.ID.Sequence == 0 || slot.order.ID != state.ID || state.Filled < slot.filled {
		return
	}
	quantity, value := state.Filled-slot.filled, state.Value-slot.value
	if side == Buy {
		a.cash -= value
		market.totals.Inventory += quantity
		market.totals.Buys += quantity
	} else {
		a.cash += value
		market.totals.Inventory -= quantity
		market.totals.Sells += quantity
	}
	a.reserved -= slot.reserved
	slot.filled, slot.value, slot.pending, slot.reserved = state.Filled, state.Value, false, 0
	if state.Open && side == Buy {
		slot.reserved = (slot.order.Quantity - state.Filled) * slot.order.Price
		a.reserved += slot.reserved
	}
	if !state.Open {
		*slot = orderSlot{}
	}
}

// currentOrder copies one order for a venue call outside the account lock.
func (a *account) currentOrder(index int, side Side) Order {
	a.mu.Lock()
	defer a.mu.Unlock()
	if side == Buy {
		return a.markets[index].buy.order
	}
	return a.markets[index].sell.order
}

// canceled counts a confirmed cancellation request for one outstanding order ID.
func (a *account) canceled(index int) {
	a.mu.Lock()
	a.markets[index].totals.Cancels++
	a.mu.Unlock()
}

// setProblem retains the last unresolved worker error for reports and Stop.
func (a *account) setProblem(index int, err error) {
	a.mu.Lock()
	a.markets[index].problem = err
	a.mu.Unlock()
}

// quote copies current order fields under the caller's account lock.
func (a *account) quote(index int) Quote {
	market := &a.markets[index]
	result := market.totals.Quote
	result.Bid, result.Ask = market.buy.order.Price, market.sell.order.Price
	result.BidID, result.AskID = market.buy.order.ID, market.sell.order.ID
	result.BidPending, result.AskPending = market.buy.pending, market.sell.pending
	return result
}

// readQuote returns current order data without exposing mutable slots.
func (a *account) readQuote(index int) Quote {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.quote(index)
}

// read returns one complete account result without sharing mutable state.
func (a *account) read() Totals {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.totals()
}

// totals reads fields under the caller's account lock.
func (a *account) totals() Totals {
	result := Totals{MaxReservedCash: a.maxReservedCash, Cash: a.cash, Reserved: a.reserved}
	for _, market := range a.markets {
		result.Inventory += market.totals.Inventory
		result.Events += market.totals.Events
		result.Orders += market.totals.Orders
		result.Buys += market.totals.Buys
		result.Sells += market.totals.Sells
		result.Cancels += market.totals.Cancels
	}
	return result
}

// hasOrders checks whether a stopped account still owns unresolved order IDs.
func (a *account) hasOrders() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, market := range a.markets {
		if market.buy.order.ID.Sequence != 0 || market.sell.order.ID.Sequence != 0 {
			return true
		}
	}
	return false
}
