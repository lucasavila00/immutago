//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

// Package shipping calculates delivery prices from a merchant's rate catalog.
package shipping

import (
	"fmt"
	"strings"

	"github.com/lucasavila00/immutago/pkg/im"
)

// Catalog holds one complete set of delivery rates in one currency.
type Catalog struct {
	im.Frozen
	Version      string
	Currency     string
	Destinations im.FrozenMap[string, string]
	Services     im.FrozenSlice[CarrierService]
}

// CarrierService supplies weight bands for each zone that it serves.
type CarrierService struct {
	im.Frozen
	ID           string
	Carrier      string
	Name         string
	DeliveryDays int
	Rates        im.FrozenMap[string, im.FrozenSlice[Band]]
}

// Band charges PriceCents for a parcel up to and including UpToGrams.
type Band struct {
	im.Frozen
	UpToGrams  int64
	PriceCents int64
}

// catalogDocument is the editable JSON format before validation and construction.
type catalogDocument struct {
	Version  string              `json:"version"`
	Currency string              `json:"currency"`
	Zones    map[string][]string `json:"zones"`
	Services []serviceDocument   `json:"services"`
}

// serviceDocument holds the rates that a merchant sets for one delivery service.
type serviceDocument struct {
	ID           string                    `json:"id"`
	Carrier      string                    `json:"carrier"`
	Name         string                    `json:"name"`
	DeliveryDays int                       `json:"delivery_days"`
	Rates        map[string][]bandDocument `json:"rates"`
}

// bandDocument distinguishes a missing price from an explicit free delivery band.
type bandDocument struct {
	UpToGrams  int64  `json:"up_to_grams"`
	PriceCents *int64 `json:"price_cents"`
}

// validateCatalog checks the complete document before any rates are published.
func validateCatalog(document catalogDocument) error {
	if strings.TrimSpace(document.Version) == "" {
		return fmt.Errorf("version must not be empty")
	}
	if !upperCode(document.Currency, 3) {
		return fmt.Errorf("currency must have three uppercase letters")
	}
	if err := validateZones(document.Zones); err != nil {
		return err
	}
	if len(document.Services) == 0 {
		return fmt.Errorf("catalog needs at least one service")
	}
	ids := make(map[string]bool)
	for _, service := range document.Services {
		if ids[service.ID] {
			return fmt.Errorf("duplicate service ID %q", service.ID)
		}
		ids[service.ID] = true
		if err := validateService(service, document.Zones); err != nil {
			return fmt.Errorf("service %q: %w", service.ID, err)
		}
	}
	return nil
}

// validateZones requires each destination country to belong to exactly one zone.
func validateZones(zones map[string][]string) error {
	if len(zones) == 0 {
		return fmt.Errorf("catalog needs at least one zone")
	}
	countries := make(map[string]bool)
	for zone, members := range zones {
		if strings.TrimSpace(zone) == "" || len(members) == 0 {
			return fmt.Errorf("each zone needs a name and at least one country")
		}
		for _, country := range members {
			if !upperCode(country, 2) {
				return fmt.Errorf("country %q must have two uppercase letters", country)
			}
			if countries[country] {
				return fmt.Errorf("country %q occurs more than once", country)
			}
			countries[country] = true
		}
	}
	return nil
}

// validateService checks the service details and its references to named zones.
func validateService(service serviceDocument, zones map[string][]string) error {
	if strings.TrimSpace(service.ID) == "" || strings.TrimSpace(service.Carrier) == "" ||
		strings.TrimSpace(service.Name) == "" {
		return fmt.Errorf("ID, carrier, and name must not be empty")
	}
	if service.DeliveryDays <= 0 {
		return fmt.Errorf("delivery_days must be positive")
	}
	if len(service.Rates) == 0 {
		return fmt.Errorf("rates must not be empty")
	}
	for zone, bands := range service.Rates {
		if _, found := zones[zone]; !found {
			return fmt.Errorf("unknown zone %q", zone)
		}
		if err := validateBands(bands); err != nil {
			return fmt.Errorf("zone %q: %w", zone, err)
		}
	}
	return nil
}

// validateBands requires increasing upper weights and explicit nonnegative prices.
func validateBands(bands []bandDocument) error {
	if len(bands) == 0 {
		return fmt.Errorf("at least one weight band is required")
	}
	var previous int64
	for _, band := range bands {
		if band.UpToGrams <= previous {
			return fmt.Errorf("up_to_grams must be positive and strictly increasing")
		}
		if band.PriceCents == nil || *band.PriceCents < 0 {
			return fmt.Errorf("price_cents must be present and nonnegative")
		}
		previous = band.UpToGrams
	}
	return nil
}

// upperCode checks the spelling of a country or currency code.
func upperCode(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, letter := range value {
		if letter < 'A' || letter > 'Z' {
			return false
		}
	}
	return true
}

// buildCatalog constructs immutable records from a document that passed validation.
func buildCatalog(document catalogDocument) Catalog {
	destinations := make(map[string]string)
	for zone, countries := range document.Zones {
		for _, country := range countries {
			destinations[country] = zone
		}
	}
	frozenDestinations := im.FreezeMap(destinations)
	services := make([]CarrierService, 0, len(document.Services))
	for _, source := range document.Services {
		rates := make(map[string]im.FrozenSlice[Band], len(source.Rates))
		for zone, sourceBands := range source.Rates {
			bands := make([]Band, 0, len(sourceBands))
			for _, sourceBand := range sourceBands {
				bands = append(bands, Band{UpToGrams: sourceBand.UpToGrams, PriceCents: *sourceBand.PriceCents})
			}
			frozenBands := im.FreezeSlice(bands)
			rates[zone] = frozenBands
		}
		frozenRates := im.FreezeMap(rates)
		services = append(services, CarrierService{
			ID: source.ID, Carrier: source.Carrier, Name: source.Name,
			DeliveryDays: source.DeliveryDays, Rates: frozenRates,
		})
	}
	frozenServices := im.FreezeSlice(services)
	return Catalog{
		Version: document.Version, Currency: document.Currency,
		Destinations: frozenDestinations, Services: frozenServices,
	}
}
