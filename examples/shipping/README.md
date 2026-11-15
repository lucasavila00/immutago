# Delivery quotes

A merchant keeps a local catalog of destination zones, carrier services, and
weight bands. This server quotes a shipment with one or more parcels. Each offer
uses one service for the whole shipment and shows the price for each parcel.

Run from the repository root:

```sh
go run ./examples/shipping/cmd/shipping
```

In another terminal, request a quote:

```sh
curl -s http://127.0.0.1:8080/quotes \
  -H 'Content-Type: application/json' \
  -d '{"destination":"US","parcels":[{"weight_grams":1000},{"weight_grams":1001}]}'
```

The [sample catalog](catalog.json) gives this response, shown with indentation:

```json
{
  "catalog_version": "2026-11",
  "currency": "USD",
  "destination": "US",
  "zone": "domestic",
  "quotes": [
    {
      "service_id": "parcel-ground",
      "carrier": "Parcel Post",
      "name": "Ground",
      "delivery_days": 5,
      "parcel_cents": [650, 1000],
      "total_cents": 1650
    },
    {
      "service_id": "city-express",
      "carrier": "City Courier",
      "name": "Express",
      "delivery_days": 2,
      "parcel_cents": [1200, 2200],
      "total_cents": 3400
    }
  ]
}
```

Offers are ordered by total price. Equal prices keep catalog order.
`parcel_cents` keeps the request's parcel order. Weights are whole grams; prices
are whole cents in the catalog's currency. `delivery_days` is the merchant's
estimate for that service.

The first weight band includes all positive weights up to its limit. Each later
band starts above the previous limit and includes its own limit. A service is
eligible only when its zone rates cover every parcel. For example, a 10,001 g
parcel to US can use Ground but cannot use Express.

A valid request with no eligible service returns HTTP 200 and `quotes: []`.
`no_match` explains the result:

| Value | Meaning |
| --- | --- |
| `destination_not_served` | The country is absent from the catalog's zones. |
| `no_service_for_all_parcels` | The country has a zone, but no service covers all parcel weights. |

Requests need a two-letter uppercase destination and at least one positive
parcel weight. Invalid requests return HTTP 400 with an `error` message.
POST is the only method on `/quotes`.

## Change the rates

Edit [catalog.json](catalog.json) while the server runs. Change `version` when
you change the rates. For example, set the version to `2026-12` and the first
Ground band's price to 750. Repeat the request after one second. Ground will
cost 1750 cents, and the response will name the new version.

Each country must occur in exactly one zone. Service IDs must be unique.
Each service needs a carrier, name, positive delivery estimate, and rates for
known zones. Each zone's bands must have positive, strictly increasing weight
limits and explicit nonnegative prices. A zero price permits free delivery.
The currency has three uppercase letters. All services use that currency.
Unknown JSON fields and extra JSON documents are rejected.

The loader checks the complete file before publishing it. An invalid edit or a
file-read error is logged; requests continue to use the last valid catalog.
A corrected file becomes available on the next reload. A missing or invalid
initial catalog prevents startup. Every quote selects one catalog version,
including when the file changes during the calculation.

Use `-catalog path/to/rates.json` to select another catalog, or
`-addr 127.0.0.1:8090` to select another listen address. Press Ctrl+C to stop. The command
lets active HTTP requests finish before it stops the catalog loader.

## Read the code

| Source | Responsibility |
| --- | --- |
| [http.go](http.go) | Read a request, select one catalog, calculate the quote, and write JSON. |
| [quote.go](quote.go) | Select zones and bands, price every parcel, and order the offers. |
| [catalog.go](catalog.go) | Validate the editable document and construct immutable rates. |
| [loader.go](loader.go) | Load before startup, poll the file, and publish complete catalogs. |
| [cmd/shipping/main.go](cmd/shipping/main.go) | Start the service and HTTP server; handle shutdown. |

The catalog contains Frozen records and collections. `CatalogCell` publishes a
whole replacement. `CatalogProjection` lets a quote keep one selected version.
The response copies the text it needs into ordinary Go data for JSON encoding.
The only supervised service is the catalog loader.

The [quote tests](quote_test.go) check totals, weight limits, sorting, and no-match
results. The [catalog tests](catalog_test.go) check merchant input. The
[loader tests](loader_test.go) check failed edits, recovery, startup, shutdown,
and concurrent calculations. The [HTTP tests](http_test.go) and
[command tests](cmd/shipping/main_test.go) check requests and server shutdown.

After a Frozen record changes, regenerate its projections:

```sh
go generate ./examples/shipping
go test -race ./examples/shipping/...
go run ./cmd/immutago ./examples/shipping/...
```
