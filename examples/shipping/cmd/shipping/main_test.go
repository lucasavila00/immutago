package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/lucasavila00/immutago/examples/shipping"
)

// TestServerQuoteAndShutdown starts the command, sends an HTTP request, and cancels it.
func TestServerQuoteAndShutdown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- run(ctx, "../../catalog.json", listener) }()
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Post("http://"+listener.Addr().String()+"/quotes", "application/json", strings.NewReader(
		`{"destination":"US","parcels":[{"weight_grams":1000},{"weight_grams":1001}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	var quote shipping.Response
	if err := json.NewDecoder(response.Body).Decode(&quote); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || len(quote.Quotes) != 2 || quote.Quotes[0].TotalCents != 1650 {
		t.Fatalf("HTTP quote = %d, %+v", response.StatusCode, quote)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP server did not stop")
	}
}

// TestServerRejectsMissingCatalog checks startup failure and release of the listen socket.
func TestServerRejectsMissingCatalog(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if err := run(context.Background(), "missing-catalog.json", listener); err == nil {
		t.Fatal("server started without a catalog")
	}
	if connection, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second); err == nil {
		_ = connection.Close()
		t.Fatal("listen socket remained open after startup failed")
	}
}
