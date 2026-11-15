# HFT market-making simulator

This example will replace the rollout log monitor. It models a local trial of
two market-making strategies. Each strategy receives the same market updates
and uses its own explicit paper budget. A market update can fill a previous
one-unit quote, then produce new quotes from price, inventory, and risk limits.
All prices and cash use integer cents. There is no broker connection.

The first checkpoint contains the price-to-quote calculation and tests for
shared cash reservations, simulated fills, replay, and cancellation. The service
tree, command, sample inputs, lifecycle tests, and cost comparisons are in progress.

The planned service tree has a policy loader beside a trading manager. The
manager owns active and candidate strategy groups. Each strategy owns one maker
worker per instrument. A worker reads its market stream and owns its resting
paper quotes. Ending a trial closes those streams and cancels those quotes.

An account belongs to the selected trial and survives service retries. Retried
workers skip already applied event sequences. An explicit removal releases that
trial's account; a later trial starts with a new paper budget. A process restart
starts a new simulation. This example will not add a database or a broker API.
