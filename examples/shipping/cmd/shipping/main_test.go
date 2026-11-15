package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
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

// TestShutdownClosesIncompleteRequest checks the fallback after the graceful deadline.
func TestShutdownClosesIncompleteRequest(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- run(ctx, "../../catalog.json", listener) }()
	connection, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = connection.Close() }()
	if err := connection.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
		t.Fatal(err)
	}
	headers := "POST /quotes HTTP/1.1\r\nHost: localhost\r\nContent-Length: 100\r\n" +
		"Expect: 100-continue\r\nContent-Type: application/json\r\n\r\n"
	if _, err := io.WriteString(connection, headers); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(connection)
	response, err := http.ReadResponse(reader, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusContinue {
		t.Fatalf("status = %d", response.StatusCode)
	}
	// The handler is reading the body. Keep it incomplete while shutdown waits.
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("shutdown error = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("shutdown did not finish")
	}
	if _, err := reader.ReadByte(); err == nil {
		t.Fatal("connection remained open after shutdown")
	} else {
		var networkError net.Error
		if errors.As(err, &networkError) && networkError.Timeout() {
			t.Fatal("connection remained open until its read deadline")
		}
	}
}
