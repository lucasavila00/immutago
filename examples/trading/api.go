package trading

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
)

// API supplies confirmed venue state for dedicated accounts with no external writers.
// Place is idempotent by Order.ID. Cancel confirms a terminal order state, including
// all fills. A zero-fill terminal result confirms that the ID was never accepted
// or was canceled without a fill; an ambiguous "not found" must be an error.
// An error can have an unknown outcome. Methods must return when ctx is canceled or its deadline expires.
// Normalized prices are positive cents below 1e9; quantities are whole units below 1e6.
// Cumulative fill quantities and values never decrease. Execution has no fees:
// buy value cannot exceed its limit times filled quantity; sell value cannot be less.
// Holdings are nonnegative and no external cash or inventory changes occur while
// the bot owns the account. These are adapter contracts, not checks proved by im.
// A live adapter and recovery after a complete process crash are outside this example.
type API interface {
	Account(context.Context, string) (Balance, error)
	Subscribe(context.Context, string, string) (Stream, error)
	Place(context.Context, string, string, Order) (OrderState, error)
	Cancel(context.Context, string, OrderID) (OrderState, error)
}

// Balance is the venue's current cash and holdings. Admission requires no open orders.
type Balance struct {
	Cash       int64
	Positions  map[string]int64
	OpenOrders int
}

// Stream sends prices and order states for one account and instrument.
// Next and Close must return when ctx ends. Close releases the market-data subscription.
type Stream interface {
	Next(context.Context) (Event, error)
	Close(context.Context) error
}

// OrderID uses one random prefix per bot start and one increasing counter retained for the complete bot run.
type OrderID struct {
	im.Copied
	Session  uint64 `json:"session"`
	Sequence uint64 `json:"sequence"`
}

// Side identifies a buy or sell order.
type Side uint8

// Buy and Sell identify the two order sides.
const (
	Buy Side = iota + 1
	Sell
)

// Order is one limit order. The example submits one unit per side.
type Order struct {
	im.Copied
	ID       OrderID `json:"id"`
	Side     Side    `json:"side"`
	Price    int64   `json:"price"`
	Quantity int64   `json:"quantity"`
}

// OrderState reports cumulative fills, their total value, and whether an order is open.
// A terminal state includes all fills; later duplicate states can be ignored.
type OrderState struct {
	im.Copied
	ID     OrderID `json:"id"`
	Filled int64   `json:"filled"`
	Value  int64   `json:"value"`
	Open   bool    `json:"open"`
}

// Tick is one normalized market update with an increasing sequence per instrument.
type Tick struct {
	im.Copied
	Sequence uint64 `json:"sequence"`
	Bid      int64  `json:"bid"`
	Ask      int64  `json:"ask"`
}

// EventKind distinguishes prices from order acknowledgments and fills.
type EventKind uint8

// PriceEvent and OrderEvent identify the two normalized event forms.
const (
	PriceEvent EventKind = iota + 1
	OrderEvent
)

// Event carries one normalized price update or cumulative order state.
type Event struct {
	im.Copied
	Kind  EventKind
	Tick  Tick
	Order OrderState
}
