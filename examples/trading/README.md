# HFT market maker bot

A trading team runs a market maker that quotes buy and sell prices. The bot
receives market prices and fill reports, adjusts its quotes for inventory,
checks its available cash, and places or cancels limit orders through a small
trading API. Each instrument worker owns its subscription and two order slots.
The example submits one unit per side and does not sell short.

The team can start a candidate strategy on a separate funded account, compare
its orders and holdings with the active strategy, then select the candidate.
Selection keeps the candidate's existing account state and workers. The old
workers unsubscribe and cancel their orders. Existing holdings remain at the
venue; the bot does not liquidate them on shutdown.

The host supplies the API adapter to `Start`. No live adapter or live trading
is part of this example. The deterministic fake exists only in tests and drives
the complete bot through market events, fills, failures, and cleanup.

## State and ownership

The policy loader reads `policy.json` beside the trading service tree. Trading
owns active and optional candidate strategy groups. Each Strategy owns a Maker
worker per instrument and publishes periodic immutable reports. A Maker's
background loop reads its market subscription and calls the trading API.

Account state survives automatic service retries. Cash is reserved before an
order is placed. A lost placement response keeps the reservation and the same
client ID. Retry first cancels and reconciles that ID. Cumulative fill values
prevent a repeated acknowledgment from spending cash twice. A confirmed cancel
includes all fills before unused cash is released.

Accounts must be dedicated to this bot while selected. Admission rejects an
account with open orders. Active and candidate use different funded accounts.
A later admission reads actual venue balances. Recovery after a full process
crash is outside this example. Cancellation has a bounded deadline; a failure
keeps the unresolved IDs and reservations, reports an error, and stops the bot.

The adapter contract uses fee-free limit executions and no external account
changes. It must honor context deadlines, idempotent client IDs, and final
cumulative fills on cancellation. These are ordinary Go dependency contracts.
The im checker does not prove exchange behavior or service isolation.

## Work in progress

The API, bot, strategy selection, numeric risk calculation, and first complete
workflow test are implemented. Further lifecycle tests, cost tests, benchmark
comparison, host usage, and removal of the old rollout example are in progress.
