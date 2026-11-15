package shipping

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestCatalogValidation checks errors that would make a merchant's prices ambiguous or incomplete.
func TestCatalogValidation(t *testing.T) {
	cases := []struct {
		name   string
		change func(*catalogDocument)
		want   string
	}{
		{"version", func(d *catalogDocument) { d.Version = " " }, "version"},
		{"currency", func(d *catalogDocument) { d.Currency = "usd" }, "currency"},
		{"zones", func(d *catalogDocument) { d.Zones = nil }, "at least one zone"},
		{"empty zone", func(d *catalogDocument) { d.Zones["domestic"] = nil }, "at least one country"},
		{"country spelling", func(d *catalogDocument) { d.Zones["domestic"] = []string{"us"} }, "uppercase"},
		{"duplicate country", func(d *catalogDocument) { d.Zones["nearby"] = []string{"US"} }, "more than once"},
		{"services", func(d *catalogDocument) { d.Services = nil }, "at least one service"},
		{"duplicate service", func(d *catalogDocument) { d.Services[1].ID = d.Services[0].ID }, "duplicate"},
		{"service ID", func(d *catalogDocument) { d.Services[0].ID = "" }, "must not be empty"},
		{"carrier", func(d *catalogDocument) { d.Services[0].Carrier = "" }, "must not be empty"},
		{"name", func(d *catalogDocument) { d.Services[0].Name = "" }, "must not be empty"},
		{"delivery days", func(d *catalogDocument) { d.Services[0].DeliveryDays = 0 }, "must be positive"},
		{"rates", func(d *catalogDocument) { d.Services[0].Rates = nil }, "rates must not be empty"},
		{"unknown zone", func(d *catalogDocument) { d.Services[0].Rates["unknown"] = nil }, "unknown zone"},
		{"empty bands", func(d *catalogDocument) { d.Services[0].Rates["domestic"] = nil }, "weight band"},
		{"zero limit", func(d *catalogDocument) { d.Services[0].Rates["domestic"][0].UpToGrams = 0 }, "increasing"},
		{"duplicate limit", func(d *catalogDocument) {
			d.Services[0].Rates["domestic"][1].UpToGrams = 1000
		}, "increasing"},
		{"decreasing limit", func(d *catalogDocument) {
			d.Services[0].Rates["domestic"][1].UpToGrams = 500
		}, "increasing"},
		{"missing price", func(d *catalogDocument) {
			d.Services[0].Rates["domestic"][0].PriceCents = nil
		}, "price_cents"},
		{"negative price", func(d *catalogDocument) {
			d.Services[0].Rates["domestic"][0].PriceCents = new(int64(-1))
		}, "nonnegative"},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			document := sampleDocument(t)
			item.change(&document)
			err := validateCatalog(document)
			if err == nil || !strings.Contains(err.Error(), item.want) {
				t.Fatalf("validation error = %v; want %q", err, item.want)
			}
		})
	}
}

// sampleDocument decodes the editable sample without constructing protected data.
func sampleDocument(t *testing.T) catalogDocument {
	t.Helper()
	data := sampleJSON(t)
	var document catalogDocument
	if err := decodeJSON(bytes.NewReader(data), &document); err != nil {
		t.Fatal(err)
	}
	return document
}

// sampleJSON reads the catalog used by the command and the application tests.
func sampleJSON(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}
