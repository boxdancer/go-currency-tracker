package client_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/boxdancer/go-currency-tracker/internal/client"
)

// Table-driven tests for CoinGeckoClient.GetPrice.
func TestCoinGeckoClient_GetPrice(t *testing.T) {
	tests := []struct {
		name              string
		handler           http.HandlerFunc
		clientTimeout     time.Duration // passed to NewCoinGeckoClient
		ctxTimeout        time.Duration // if >0 create context with timeout
		want              float64
		wantErr           bool
		assertErrContains string // substring to assert in error (case-insensitive)
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"bitcoin":{"usd":123.45}}`))
			},
			clientTimeout: 5 * time.Second,
			ctxTimeout:    0,
			want:          123.45,
			wantErr:       false,
		},
		{
			name: "non-200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "boom", http.StatusInternalServerError)
			},
			clientTimeout:     5 * time.Second,
			wantErr:           true,
			assertErrContains: "unexpected status",
		},
		{
			name: "bad json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("{not-json}"))
			},
			clientTimeout:     5 * time.Second,
			wantErr:           true,
			assertErrContains: "decode json",
		},
		{
			name: "missing id",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"other":{"usd":1}}`))
			},
			clientTimeout:     5 * time.Second,
			wantErr:           true,
			assertErrContains: "no id",
		},
		{
			name: "missing vs",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"bitcoin":{"eur":1}}`))
			},
			clientTimeout:     5 * time.Second,
			wantErr:           true,
			assertErrContains: "no vs",
		},
		{
			name: "context timeout",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// server delays; but request uses ctx with small timeout
				time.Sleep(200 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"bitcoin":{"usd":1}}`))
			},
			clientTimeout:     1 * time.Second,
			ctxTimeout:        50 * time.Millisecond,
			wantErr:           true,
			assertErrContains: "context",
		},
		{
			name: "client timeout",
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(200 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"bitcoin":{"usd":1}}`))
			},
			clientTimeout:     50 * time.Millisecond, // http.Client Timeout
			ctxTimeout:        0,
			wantErr:           true,
			assertErrContains: "timeout",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// start mock server with handler
			ts := httptest.NewServer(tc.handler)
			defer ts.Close()

			// create client and point it to test server
			c := client.NewCoinGeckoClient(tc.clientTimeout)
			c.SetBaseURL(ts.URL)

			// build context
			var ctx context.Context
			var cancel context.CancelFunc
			if tc.ctxTimeout > 0 {
				ctx, cancel = context.WithTimeout(context.Background(), tc.ctxTimeout)
			} else {
				ctx, cancel = context.WithCancel(context.Background())
			}
			defer cancel()

			got, err := c.GetPrice(ctx, "bitcoin", "usd")

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				// special-case: for client-timeout assert net.Error.Timeout()
				if tc.name == "client timeout" {
					// Разрешаем два сценария: либо net.Error с Timeout(),
					// либо стандартная ошибка дедлайна контекста.
					if ne, ok := err.(net.Error); ok && ne.Timeout() {
						return
					}
					if errors.Is(err, context.DeadlineExceeded) {
						return
					}
					t.Fatalf("expected timeout error, got: %v", err)
				}
				// otherwise, check substring if provided
				if tc.assertErrContains != "" {
					if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.assertErrContains)) &&
						!errorsIsContextDeadline(err) {
						t.Fatalf("expected error containing %q (or context deadline), got: %v", tc.assertErrContains, err)
					}
				}
				return
			}

			// want no error
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("wrong price: want %v got %v", tc.want, got)
			}
		})
	}
}

// helper: some http client errors for context deadlines come in different shapes; check for context deadline
func errorsIsContextDeadline(err error) bool {
	if err == nil {
		return false
	}
	// quick check using error string and standard type
	if strings.Contains(strings.ToLower(err.Error()), "context") || strings.Contains(strings.ToLower(err.Error()), "deadline") {
		return true
	}
	return false
}
