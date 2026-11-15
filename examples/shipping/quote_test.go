package shipping

import (
	"math"
	"reflect"
	"testing"

	"github.com/lucasavila00/immutago/pkg/im"
)

// TestMultiParcelQuote checks the complete offer data for two parcels.
func TestMultiParcelQuote(t *testing.T) {
	catalog := sampleCatalog(t)
	response := Quote(catalog, Request{
		Destination: "US", Parcels: []Parcel{{WeightGrams: 1000}, {WeightGrams: 1001}},
	})
	want := Response{
		CatalogVersion: "2026-11", Currency: "USD", Destination: "US", Zone: "domestic",
		Quotes: []Offer{
			{ServiceID: "parcel-ground", Carrier: "Parcel Post", Name: "Ground", DeliveryDays: 5,
				ParcelCents: []int64{650, 1000}, TotalCents: 1650},
			{ServiceID: "city-express", Carrier: "City Courier", Name: "Express", DeliveryDays: 2,
				ParcelCents: []int64{1200, 2200}, TotalCents: 3400},
		},
	}
	if !reflect.DeepEqual(response, want) {
		t.Fatalf("quote = %+v; want %+v", response, want)
	}
}

// TestBandCopy changes an independent numeric band without changing the selected catalog.
func TestBandCopy(t *testing.T) {
	catalog := sampleCatalog(t)
	service := NewCarrierServiceProjection(catalog.Services().AtPointer(0))
	value, found := service.Rates().Get("domestic")
	if !found {
		t.Fatal("missing sample rates")
	}
	bands := im.SliceOf(value)
	band := im.Value(bands.At(0))
	price := band.PriceCents
	band.PriceCents += 25
	if band.PriceCents != price+25 || im.Value(bands.At(0)).PriceCents != price {
		t.Fatal("band copy did not remain independent")
	}
}

// TestWeightBands checks inclusive band edges, destination zones, and service limits.
func TestWeightBands(t *testing.T) {
	cases := []struct {
		name        string
		destination string
		weight      int64
		prices      []int64
		reason      string
	}{
		{"minimum", "US", 1, []int64{650, 1200}, ""},
		{"first edge", "US", 1000, []int64{650, 1200}, ""},
		{"above first edge", "US", 1001, []int64{1000, 2200}, ""},
		{"second edge", "US", 5000, []int64{1000, 2200}, ""},
		{"above second edge", "US", 5001, []int64{1800, 2200}, ""},
		{"express limit", "US", 10000, []int64{1800, 2200}, ""},
		{"above express limit", "US", 10001, []int64{1800}, ""},
		{"ground limit", "US", 20000, []int64{1800}, ""},
		{"too heavy", "US", 20001, nil, "no_service_for_all_parcels"},
		{"nearby", "CA", 1000, []int64{1400}, ""},
		{"nearby edge", "MX", 5000, []int64{2500}, ""},
		{"nearby limit", "CA", 5001, nil, "no_service_for_all_parcels"},
		{"unknown destination", "GB", 1000, nil, "destination_not_served"},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			catalog := sampleCatalog(t)
			response := Quote(catalog, Request{
				Destination: item.destination, Parcels: []Parcel{{WeightGrams: item.weight}},
			})
			var prices []int64
			for _, offer := range response.Quotes {
				prices = append(prices, offer.TotalCents)
			}
			if response.Error != "" || response.NoMatch != item.reason || !reflect.DeepEqual(prices, item.prices) {
				t.Fatalf("quote = %+v; want prices %v and reason %q", response, item.prices, item.reason)
			}
		})
	}
}

// TestAllParcelsMustFit prevents an offer that prices only part of a shipment.
func TestAllParcelsMustFit(t *testing.T) {
	catalog := sampleCatalog(t)
	response := Quote(catalog, Request{
		Destination: "US", Parcels: []Parcel{{WeightGrams: 500}, {WeightGrams: 20001}},
	})
	if len(response.Quotes) != 0 || response.NoMatch != "no_service_for_all_parcels" {
		t.Fatalf("partial shipment received an offer: %+v", response)
	}
}

// TestOfferOrder checks price ordering and an explicit zero-price band.
func TestOfferOrder(t *testing.T) {
	document := sampleDocument(t)
	document.Services[1].Rates["domestic"][0].PriceCents = new(int64(0))
	loader := &Loader{}
	loader.current.Store(buildCatalog(document))
	response := Quote(loader.Catalog(), Request{Destination: "US", Parcels: []Parcel{{WeightGrams: 500}}})
	if len(response.Quotes) != 2 || response.Quotes[0].ServiceID != "city-express" ||
		response.Quotes[0].TotalCents != 0 || response.Quotes[1].TotalCents != 650 {
		t.Fatalf("offers are not in price order: %+v", response)
	}
}

// TestQuoteRejectsOverflow prevents a wrapped total from appearing as a cheap offer.
func TestQuoteRejectsOverflow(t *testing.T) {
	document := sampleDocument(t)
	document.Services[0].Rates["domestic"][0].PriceCents = new(int64(math.MaxInt64))
	loader := &Loader{}
	loader.current.Store(buildCatalog(document))
	response := Quote(loader.Catalog(), Request{
		Destination: "US", Parcels: []Parcel{{WeightGrams: 500}, {WeightGrams: 500}},
	})
	if response.Error == "" || len(response.Quotes) != 0 {
		t.Fatalf("overflow produced an offer: %+v", response)
	}
	response = Quote(loader.Catalog(), Request{
		Destination: "US", Parcels: []Parcel{{WeightGrams: 500}, {WeightGrams: 500}, {WeightGrams: 20001}},
	})
	if response.Error != "" || response.NoMatch != "no_service_for_all_parcels" {
		t.Fatalf("ineligible service produced a price error: %+v", response)
	}
}

// sampleCatalog loads the shipped catalog through the same path as service startup.
func sampleCatalog(t *testing.T) CatalogProjection {
	t.Helper()
	loader := &Loader{}
	if err := loader.reload("catalog.json"); err != nil {
		t.Fatal(err)
	}
	return loader.Catalog()
}
