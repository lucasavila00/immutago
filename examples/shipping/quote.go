package shipping

import (
	"fmt"
	"math"
	"sort"

	"github.com/lucasavila00/immutago/pkg/im"
)

// Request asks for delivery of all Parcels to one destination country.
type Request struct {
	Destination string   `json:"destination"`
	Parcels     []Parcel `json:"parcels"`
}

// Parcel specifies the packed weight of one parcel in whole grams.
type Parcel struct {
	WeightGrams int64 `json:"weight_grams"`
}

// Offer prices every parcel with the same carrier service.
type Offer struct {
	ServiceID    string  `json:"service_id"`
	Carrier      string  `json:"carrier"`
	Name         string  `json:"name"`
	DeliveryDays int     `json:"delivery_days"`
	ParcelCents  []int64 `json:"parcel_cents"`
	TotalCents   int64   `json:"total_cents"`
}

// Response contains prices from one catalog, or Error for an invalid request.
type Response struct {
	CatalogVersion string  `json:"catalog_version"`
	Currency       string  `json:"currency"`
	Destination    string  `json:"destination"`
	Zone           string  `json:"zone,omitempty"`
	Quotes         []Offer `json:"quotes"`
	NoMatch        string  `json:"no_match,omitempty"`
	Error          string  `json:"-"`
}

// Quote returns eligible services, cheapest first, using only the selected catalog.
func Quote(catalog CatalogProjection, request Request) Response {
	if err := validateRequest(request); err != nil {
		return Response{Error: err.Error()}
	}
	response := Response{
		CatalogVersion: catalog.Version().Clone(), Currency: catalog.Currency().Clone(),
		Destination: request.Destination, Quotes: make([]Offer, 0),
	}
	zone, found := catalog.Destinations().Get(request.Destination)
	if !found {
		response.NoMatch = "destination_not_served"
		return response
	}
	response.Zone = im.StringOf(zone).Clone()
	services := catalog.Services()
	for index := 0; index < services.Len(); index++ {
		service := NewCarrierServiceProjection(services.AtPointer(index))
		rate, found := service.Rates().Get(response.Zone)
		if !found {
			continue
		}
		bands := im.SliceOf(rate)
		prices := make([]int64, 0, len(request.Parcels))
		for _, parcel := range request.Parcels {
			price := parcelPrice(bands, parcel.WeightGrams)
			if price < 0 {
				break
			}
			prices = append(prices, price)
		}
		if len(prices) != len(request.Parcels) {
			continue
		}
		var total int64
		for _, price := range prices {
			if price > math.MaxInt64-total {
				return Response{Error: "total price exceeds the supported integer range"}
			}
			total += price
		}
		response.Quotes = append(response.Quotes, Offer{
			ServiceID: service.ID().Clone(), Carrier: service.Carrier().Clone(), Name: service.Name().Clone(),
			DeliveryDays: service.DeliveryDays(), ParcelCents: prices, TotalCents: total,
		})
	}
	if len(response.Quotes) == 0 {
		response.NoMatch = "no_service_for_all_parcels"
	}
	sort.SliceStable(response.Quotes, func(i, j int) bool {
		return response.Quotes[i].TotalCents < response.Quotes[j].TotalCents
	})
	return response
}

// validateRequest rejects missing destinations, empty shipments, and invalid weights.
func validateRequest(request Request) error {
	if !upperCode(request.Destination, 2) {
		return fmt.Errorf("destination must have two uppercase letters")
	}
	if len(request.Parcels) == 0 {
		return fmt.Errorf("at least one parcel is required")
	}
	for index, parcel := range request.Parcels {
		if parcel.WeightGrams <= 0 {
			return fmt.Errorf("parcel %d: weight_grams must be positive", index+1)
		}
	}
	return nil
}

// parcelPrice finds the first inclusive weight band, or returns -1 if none fits.
func parcelPrice(bands im.SliceProjection[Band], weight int64) int64 {
	for index := 0; index < bands.Len(); index++ {
		band := NewBandProjection(bands.AtPointer(index))
		if weight <= band.UpToGrams() {
			return band.PriceCents()
		}
	}
	return -1
}
