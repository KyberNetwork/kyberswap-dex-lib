package limitorder

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
)

const liveBaseURL = "https://limit-order.kyberswap.com"

// TestGetOpSignatures_LiveAPI tests against the actual limit-order.kyberswap.com API.
func TestGetOpSignatures_LiveAPI(t *testing.T) {
	test.SkipCI(t)
	c := NewHTTPClient(liveBaseURL)

	t.Run("valid chain with nonexistent orders returns empty list without error", func(t *testing.T) {
		sigs, err := c.GetOpSignatures(t.Context(), 1, []int64{999999999})
		require.NoError(t, err)
		assert.Empty(t, sigs)
	})

	t.Run("invalid chainId causes HTTP 400 which wraps ErrGetOpSignaturesFailed", func(t *testing.T) {
		// chainId=0 is invalid; the live API returns HTTP 400
		_, err := c.GetOpSignatures(t.Context(), 0, []int64{1})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrGetOpSignaturesFailed),
			"expected ErrGetOpSignaturesFailed in chain, got: %v", err)
	})
}

// TestGetOpSignatures_ErrorWrapping verifies all local error paths wrap ErrGetOpSignaturesFailed.
func TestGetOpSignatures_ErrorWrapping(t *testing.T) {
	t.Run("non-zero code at HTTP 200 wraps ErrGetOpSignaturesFailed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":4001,"message":"some backend error","data":null}`))
		}))
		defer srv.Close()

		c := NewHTTPClient(srv.URL)
		_, err := c.GetOpSignatures(t.Context(), 1, []int64{123})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrGetOpSignaturesFailed), "got: %v", err)
	})

	t.Run("HTTP 4xx response wraps ErrGetOpSignaturesFailed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer srv.Close()

		c := NewHTTPClient(srv.URL)
		_, err := c.GetOpSignatures(t.Context(), 1, []int64{123})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrGetOpSignaturesFailed), "got: %v", err)
	})

	t.Run("transport/network error wraps ErrGetOpSignaturesFailed", func(t *testing.T) {
		c := NewHTTPClient("http://127.0.0.1:1")
		_, err := c.GetOpSignatures(t.Context(), 1, []int64{123})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrGetOpSignaturesFailed), "got: %v", err)
	})
}

// TestNewHTTPClientWithConfig covers the knobs the Config-taking constructor adds.
func TestNewHTTPClientWithConfig(t *testing.T) {
	t.Run("negative retry count does not disable the request", func(t *testing.T) {
		// resty takes its single-shot path only when RetryCount == 0; any other value goes
		// through Backoff, whose `attempt <= retries` loop never runs for a negative count
		// and returns a nil response. Clamping keeps that out of the caller's hands.
		var hits int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			hits++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":0,"data":{"operatorSignatures":[]}}`))
		}))
		defer srv.Close()

		c := NewHTTPClientWithConfig(&Config{LimitOrderHTTPUrl: srv.URL, HTTPRetryCount: -1})
		require.NotPanics(t, func() {
			_, err := c.GetOpSignatures(t.Context(), 1, []int64{123})
			require.NoError(t, err)
		})
		assert.Equal(t, 1, hits)
	})

	t.Run("unset timeout falls back to the default", func(t *testing.T) {
		c := NewHTTPClientWithConfig(&Config{LimitOrderHTTPUrl: "http://example.invalid"})
		assert.Equal(t, defaultHTTPTimeout, c.client.GetClient().Timeout)
	})

	t.Run("injected client shares its Transport but keeps its own Timeout", func(t *testing.T) {
		transport := &http.Transport{MaxIdleConnsPerHost: 64}
		shared := &http.Client{Transport: transport, Timeout: time.Hour}

		c := NewHTTPClientWithConfig(&Config{
			LimitOrderHTTPUrl: "http://example.invalid",
			HTTPClient:        shared,
			HTTPTimeout:       durationjson.Duration{Duration: 250 * time.Millisecond},
		})

		assert.Same(t, transport, c.client.GetClient().Transport, "connection pool must be shared")
		assert.Equal(t, 250*time.Millisecond, c.client.GetClient().Timeout)
		assert.Equal(t, time.Hour, shared.Timeout, "caller's client must not be mutated")
	})
}
