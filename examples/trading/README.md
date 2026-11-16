# HFT market maker bot

A trading team wants to change how its market maker quotes prices. It starts a
candidate strategy on a separate funded account while the active strategy keeps
working. Both receive prices for the same instruments. The operator compares
their quotes, fills, cash, and holdings, then selects the candidate or ends its
trial.

This bot does that work through a small trading API. Each instrument worker
receives market prices and fill reports, adjusts prices for inventory, checks
available cash, and places or cancels one-unit limit orders. The bot does not
sell short. Selecting the candidate keeps its existing account and workers.
The old workers unsubscribe and cancel their orders; their holdings stay at the
venue.

The host supplies a venue adapter. The [test fake](fake_api_test.go) drives the
complete bot without a live connection. A live adapter and live trading tests
are outside this example.

## Start and read the bot

Run the complete workflow test from the repository root:

```sh
go test ./examples/trading -run TestBotQuotesFillsAndPolicy -count=1
```

A host supplies an adapter and a context, then uses the [sample configuration](data):

```go
bot, err := trading.Start(trading.Options{
    API: venue,
    Files: os.DirFS("examples/trading/data"),
    Every: time.Second,
    Log: log.Default(),
})
if err != nil {
    return err
}

report := bot.Read()
active := report.Active()
fmt.Println(active.ID().Clone(), active.Totals().Cash)

// Wait for host cancellation or a bot failure, then handle cleanup errors.
select {
case <-ctx.Done():
case <-bot.Done():
}
return bot.Stop()
```

`Every` controls configuration reads and report publication. Market workers
process subscription events as they arrive. A report is an immutable snapshot;
a caller can finish reading it after its strategy stops. Report publication
and display are outside the per-event calculation.

## Trial a strategy

`trading.json` initially selects `wide` on `desk-a`. To add `tight`, use:

```json
{
  "active": {
    "id": "wide", "account": "desk-a", "max_reserved_cash": 1000,
    "parameters": {"edge": 3, "skew": 1},
    "markets": [{"symbol": "AAA"}, {"symbol": "BBB"}]
  },
  "candidate": {
    "id": "tight", "account": "desk-b", "max_reserved_cash": 1000,
    "parameters": {"edge": 1, "skew": 1},
    "markets": [{"symbol": "AAA"}, {"symbol": "BBB"}]
  }
}
```

`AAA` and `BBB` are example instrument names. The adapter supplies their actual
prices. `desk-a` and `desk-b` must identify distinct funded accounts. The bot
reads each account's actual cash and holdings before it starts that strategy.

| Field | Meaning |
| --- | --- |
| `id` | Stable name for this selected strategy. |
| `account` | Dedicated venue account, with no other writer while selected. |
| `max_reserved_cash` | Maximum cash committed to outstanding buy orders across all instruments, in cents. It is not a starting balance or a limit on total turnover. |
| `edge` | Distance in cents from the inventory-adjusted price. |
| `skew` | Price adjustment in cents for each held unit. |
| `markets` | Distinct instruments; active and candidate use the same ordered list. |

For a new market price, the bot computes the midpoint and subtracts
`inventory * skew`. Its bid is no higher than the market bid; its ask is no
lower than the market ask. If either quote must change, it cancels the old pair,
applies all final fills, then computes a new pair from current holdings.
Each order is one unit. The cash check includes reservations held by other
instrument workers on the same account.

To select the candidate, replace `active` with the unchanged candidate object
and remove `candidate`. The bot publishes the new selection before it stops the
old group. It keeps the candidate's subscriptions, balances, and order IDs.
To end only the candidate trial, remove `candidate` and keep `active` unchanged.

A selected ID and account keep their parameters and instrument list. Use a new
ID and a different account to trial a change. After an account has been removed,
a later admission reads its actual venue balances again and uses new client
order IDs. State for completed trials does not accumulate in the bot.

## Change risk limits

The independent Policy service reads `policy.json`:

```json
{"version": 1, "max_inventory": 5, "max_spread": 10}
```

| Rule | Effect on the next price update |
| --- | --- |
| `max_inventory` | Stop placing buys when this instrument's holdings reach the limit. |
| `max_spread` | Withdraw quotes when the market ask minus bid exceeds this many cents. |
| `max_reserved_cash` in the strategy | Hold no more than this amount in outstanding buy commitments, and never more than available cash. |

Each price update selects one complete immutable policy. Each reported quote
includes its `policy_version`. Policy changes leave strategy workers running.
A bad policy edit keeps the last valid limits and sets `policy_problem`.
A bad selection edit keeps the current selection and sets `problem`.

Configuration requires positive `version`, `max_inventory`, and `max_spread`.
The two limits are at most 1,000,000. `max_reserved_cash` is 1 through
1,000,000,000,000 cents; `edge` is 1 through 1,000,000; `skew` is 0 through
1,000,000. Invalid initial configuration prevents startup.

## State and service lifetimes

All services run in one Go process:

```text
root
├── Policy: refresh and publish immutable risk limits
└── Trading: apply strategy selection and publish reports
    ├── active lifetime group
    │   └── Strategy: publish periodic account reports
    │       ├── Maker AAA: subscription, event loop, bid and ask orders
    │       └── Maker BBB: subscription, event loop, bid and ask orders
    └── optional candidate lifetime group
        └── Strategy
            ├── Maker AAA
            └── Maker BBB
```

The bot owns account state outside the restartable service values. One account
lock protects cash, inventory, and reservations across its workers. Venue calls
run outside that lock. Each instrument retains at most one pending or resting
order per side. The background loop owns all event processing; callers receive
reports, not direct access to the worker's mutable state.

| Event | Result |
| --- | --- |
| One market connection fails | That Maker closes its subscription, cancels its orders, and retries. Other workers continue. |
| A Strategy fails | Its children finish cleanup before replacement workers start. Its account state remains. |
| Trading fails | Both strategy trees stop before reconstruction. Account state and the independent Policy remain. |
| Candidate is selected | Existing candidate workers continue. Old active workers and subscriptions stop. |
| A trial ends normally | Its orders are canceled and its account state is released. Venue holdings remain. |
| The bot stops | Children stop in reverse start order. Policy stops after its consumers. |

Cash is reserved before placement. A lost placement response keeps the
reservation and client ID. Retry first cancels and reconciles that ID before
opening another subscription. Cumulative fill deltas apply once. Confirmed
cancellation includes all fills before unused cash is released; late duplicate
reports for a closed ID have no effect.

Cleanup uses bounded API contexts. If cancellation is not confirmed, the bot
retains the unresolved IDs and reservations and stops. `Stop` returns the
failure, including the account, instrument, and IDs. It does not report that
those remote orders are gone. Failed staged selection uses the same cleanup
rule. The bot retains at most two unresolved retired accounts while it stops.

The [runtime](../../docs/reference/services.md) supplies retry delays and the
retry limit. Six failures within its window stop the root and terminate the
process. Restart after a complete process crash is a separate operator task.

## Venue contract

The [API](api.go) is deliberately small:

| Method | Required result |
| --- | --- |
| `Account` | Actual cash, positions, and open-order count for a dedicated account. Admission rejects preexisting open orders. |
| `Subscribe` | A stream of normalized price updates and cumulative order/fill states for one account and instrument. |
| `Place` | Idempotent placement by client order ID. An error can have an unknown outcome. |
| `Cancel` | A confirmed terminal state with all cumulative fills, including fills before a lost placement response. An ambiguous “not found” is an error. |
| Stream `Next` and `Close` | Event delivery and subscription cleanup that return when their context ends. |

All API calls must return on context cancellation or deadline. Prices use
integer cents below 1,000,000,000; quantities are nonnegative whole units below
1,000,000. Prices have `0 < bid < ask`. Price sequences increase per instrument
across reconnects; replayed prices can be ignored. Fill events are processed
independently of price sequences.

This example's adapter contract uses fee-free limit executions, no short
positions, and no external account changes while selected. Buy fills cost no
more than their limit; sell fills receive no less. Cumulative quantities and
values never decrease. A zero-fill terminal cancellation confirms that the ID
was never accepted or was canceled without a fill.

These are ordinary Go dependency contracts. A live adapter must meet them.
The im checker protects declared immutable data; it does not prove venue
behavior, service isolation, or complete process-crash recovery.

## Event-path costs and tests

The hot path is `Stream.Next` → price or fill processing → policy read → shared
risk state → order cancellation and placement. Numeric events, orders, and
results use `im.Copied`. Policy uses a Frozen record and generated projection.
The read through `supervise.Ref` and the policy Cell occurs for every price
update. There is no per-event report allocation or notification.

The [cost tests](cost_test.go) check zero steady-state allocations and actual
order/fill work. The [direct Go comparison](native_cost_test.go) uses the same
calculation, locks, order counter, stream, and fixed-storage test adapter, with
native atomic pointers for policy publication and selection. Reprice iterations
use fresh sequences and changed prices. A second case processes buy and sell
fills between price updates. Both check exact work counts after timing.

These benchmarks measure single-instrument event-loop CPU cost. They include
input delivery and the fixed-storage adapter calls. Setup, periodic reports,
network latency, and contention are outside the comparison. Use the manual
[Core performance workflow](../../.github/workflows/core-performance.yml) with
`scope: trading`. Measurements belong in the pull request, not this guide.

| Tests | Evidence |
| --- | --- |
| [Workflow](workflow_test.go) | Prices, fills, risk changes, invalid edits, shared cash, and old report reads. |
| [Lifetimes](lifecycle_test.go) | Independent retries, cleanup before replacement, promotion, account re-admission, and reverse shutdown. |
| [Failures](failure_test.go) | Lost placement responses, final fills on cancel, bounded failed cleanup, staged rollback, and admission with old orders. |
| [Retry exhaustion](exhaustion_test.go) | The real runtime terminates once, after owned resources stop. |

GitHub Actions runs the full repository tests, race checks, source checks, and
analyzer. The example changes no library or checker rules.
