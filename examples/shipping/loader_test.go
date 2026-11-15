package shipping

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise"
	"github.com/lucasavila00/immutago/pkg/supervise/supervisetest"
)

// TestReloadRetainsCatalog checks failed edits, a missing file, and recovery with new rates.
func TestReloadRetainsCatalog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.json")
	initial := sampleJSON(t)
	writeCatalog(t, path, initial)
	loader := &Loader{}
	if err := loader.reload(path); err != nil {
		t.Fatal(err)
	}
	selected := loader.Catalog()
	for _, invalid := range []string{
		`{`,
		strings.Replace(string(initial), `"currency"`, `"curreny"`, 1),
		strings.Replace(string(initial), `"price_cents": 650`, `"price_cents": -1`, 1),
	} {
		writeCatalog(t, path, []byte(invalid))
		if err := loader.reload(path); err == nil {
			t.Fatal("invalid edit replaced the catalog")
		}
		assertVersionPrice(t, loader.Catalog(), "2026-11", 1650)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := loader.reload(path); err == nil {
		t.Fatal("missing catalog did not produce an error")
	}
	assertVersionPrice(t, loader.Catalog(), "2026-11", 1650)
	writeCatalog(t, path, replacementJSON(initial))
	if err := loader.reload(path); err != nil {
		t.Fatal(err)
	}
	assertVersionPrice(t, loader.Catalog(), "2026-12", 1850)
	assertVersionPrice(t, selected, "2026-11", 1650)
}

// TestLoaderPollingAndShutdown checks the real Run loop and waits for a clean stop.
func TestLoaderPollingAndShutdown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.json")
	initial := sampleJSON(t)
	writeCatalog(t, path, initial)
	root := supervisetest.NewRoot(t)
	loader, err := supervise.Start[Loader](root, Args{Path: path, Every: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	assertVersionPrice(t, loader.Borrow().Catalog(), "2026-11", 1650)
	writeCatalog(t, path, replacementJSON(initial))
	deadline := time.Now().Add(5 * time.Second)
	for !loader.Borrow().Catalog().Version().EqualString("2026-12") {
		if time.Now().After(deadline) {
			t.Fatal("catalog was not reloaded")
		}
		time.Sleep(time.Millisecond)
	}
	assertVersionPrice(t, loader.Borrow().Catalog(), "2026-12", 1850)
	select {
	case <-root.Stop():
	case <-time.After(5 * time.Second):
		t.Fatal("catalog loader did not stop")
	}
}

// TestLoaderStartupFailure prevents an HTTP service from starting without valid rates.
func TestLoaderStartupFailure(t *testing.T) {
	initial := string(sampleJSON(t))
	cases := []struct {
		name, text string
		every      time.Duration
	}{
		{"missing file", "", time.Hour},
		{"invalid JSON", `{`, time.Hour},
		{"invalid catalog", strings.Replace(initial, `"USD"`, `"usd"`, 1), time.Hour},
		{"invalid interval", initial, 0},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "catalog.json")
			if item.text != "" {
				writeCatalog(t, path, []byte(item.text))
			}
			root := supervisetest.NewRoot(t)
			_, err := supervise.Start[Loader](root, Args{Path: path, Every: item.every})
			if err == nil {
				t.Fatal("startup succeeded without a valid catalog and interval")
			}
		})
	}
}

// TestConcurrentQuotes keeps all parcel charges and the version from one complete catalog.
func TestConcurrentQuotes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.json")
	initial := sampleJSON(t)
	replacement := replacementJSON(initial)
	writeCatalog(t, path, initial)
	loader := &Loader{}
	if err := loader.reload(path); err != nil {
		t.Fatal(err)
	}
	ready := make(chan struct{}, 4)
	done := make(chan struct{})
	failures := make(chan Response, 4)
	var readers sync.WaitGroup
	for range 4 {
		readers.Add(1)
		go readDuringReload(loader, ready, done, failures, &readers)
	}
	defer func() {
		close(done)
		readers.Wait()
		select {
		case response := <-failures:
			t.Errorf("quote mixed catalog versions: %+v", response)
		default:
		}
	}()
	for range 4 {
		<-ready
	}
	for range 20 {
		for _, data := range [][]byte{replacement, initial} {
			writeCatalog(t, path, data)
			if err := loader.reload(path); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// readDuringReload calculates quotes while the test replaces the catalog.
func readDuringReload(
	loader *Loader, ready chan<- struct{}, done <-chan struct{}, failures chan<- Response, readers *sync.WaitGroup,
) {
	defer readers.Done()
	ready <- struct{}{}
	for {
		select {
		case <-done:
			return
		default:
		}
		response := Quote(loader.Catalog(), Request{
			Destination: "US", Parcels: []Parcel{{WeightGrams: 1000}, {WeightGrams: 1001}},
		})
		if !consistentQuote(response) {
			failures <- response
			return
		}
	}
}

// consistentQuote recognizes the complete prices from either test catalog.
func consistentQuote(response Response) bool {
	if response.Error != "" || len(response.Quotes) != 2 ||
		len(response.Quotes[0].ParcelCents) != 2 || len(response.Quotes[1].ParcelCents) != 2 {
		return false
	}
	ground := response.Quotes[0]
	express := response.Quotes[1]
	if response.CatalogVersion == "2026-11" {
		return ground.ParcelCents[0] == 650 && ground.ParcelCents[1] == 1000 && ground.TotalCents == 1650 &&
			express.ParcelCents[0] == 1200 && express.ParcelCents[1] == 2200 && express.TotalCents == 3400
	}
	if response.CatalogVersion == "2026-12" {
		return ground.ParcelCents[0] == 750 && ground.ParcelCents[1] == 1100 && ground.TotalCents == 1850 &&
			express.ParcelCents[0] == 1400 && express.ParcelCents[1] == 2400 && express.TotalCents == 3800
	}
	return false
}

// assertVersionPrice checks the application result from one selected catalog.
func assertVersionPrice(t *testing.T, catalog CatalogProjection, version string, price int64) {
	t.Helper()
	response := Quote(catalog, Request{
		Destination: "US", Parcels: []Parcel{{WeightGrams: 1000}, {WeightGrams: 1001}},
	})
	if response.CatalogVersion != version || len(response.Quotes) != 2 || response.Quotes[0].TotalCents != price {
		t.Fatalf("quote = %+v; want version %s and price %d", response, version, price)
	}
}

// replacementJSON changes both services' domestic prices and the catalog version.
func replacementJSON(initial []byte) []byte {
	text := strings.Replace(string(initial), `"2026-11"`, `"2026-12"`, 1)
	text = strings.Replace(text, `"price_cents": 650`, `"price_cents": 750`, 1)
	text = strings.Replace(text, `"price_cents": 1000`, `"price_cents": 1100`, 1)
	text = strings.Replace(text, `"price_cents": 1200`, `"price_cents": 1400`, 1)
	text = strings.Replace(text, `"price_cents": 2200`, `"price_cents": 2400`, 1)
	return []byte(text)
}

// writeCatalog replaces the merchant's catalog file for the next reload.
func writeCatalog(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}
