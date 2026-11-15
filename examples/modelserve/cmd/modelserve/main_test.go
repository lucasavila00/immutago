package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestRunDrainsRequestBeforeStop keeps an admitted request open across cancellation.
func TestRunDrainsRequestBeforeStop(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- run(ctx, "../../models", address, time.Second) }()
	connection := dialServer(t, address, done)
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	body := `{"area_m2":90,"rooms":3}`
	_, err = fmt.Fprintf(connection, "POST /predict HTTP/1.1\r\nHost: localhost\r\n"+
		"Expect: 100-continue\r\nContent-Length: %d\r\n\r\n", len(body))
	if err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(connection)
	response, err := http.ReadResponse(reader, &http.Request{Method: http.MethodPost})
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != 100 {
		t.Fatal(response.StatusCode)
	}
	cancel()
	select {
	case err := <-done:
		t.Fatalf("command stopped before the request finished: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	if _, err := io.WriteString(connection, body); err != nil {
		t.Fatal(err)
	}
	response, err = http.ReadResponse(reader, &http.Request{Method: http.MethodPost})
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil || response.StatusCode != 200 || !strings.Contains(string(data), "baseline-v1") {
		t.Fatalf("response: %s %v", data, err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("command did not finish shutdown")
	}
}

// dialServer waits for startup and reports an early command failure.
func dialServer(t *testing.T, address string, done <-chan error) net.Conn {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		connection, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err == nil {
			return connection
		}
		select {
		case err := <-done:
			t.Fatalf("command failed at startup: %v", err)
		case <-time.After(10 * time.Millisecond):
		}
	}
	t.Fatal("server did not open its listener")
	return nil
}

// TestRunInitialFailure returns a model setup error before opening HTTP ingress.
func TestRunInitialFailure(t *testing.T) {
	if err := run(context.Background(), t.TempDir(), "127.0.0.1:0", time.Second); err == nil {
		t.Fatal("command accepted a missing primary manifest")
	}
}
