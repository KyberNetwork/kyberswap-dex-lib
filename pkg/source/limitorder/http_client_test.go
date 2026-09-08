package limitorder

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/go-resty/resty/v2"
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

// TestNewHTTPClientWithRestyClient covers the deprecated wrapper: it must bound a client
// that has no timeout without retuning one that already does.
func TestNewHTTPClientWithRestyClient(t *testing.T) {
	const baseURL = "http://example.invalid"

	t.Run("injected client without a timeout gets the default", func(t *testing.T) {
		c := NewHTTPClientWithRestyClient(baseURL, resty.New())
		assert.Equal(t, defaultHTTPTimeout, c.client.GetClient().Timeout)
		assert.Equal(t, baseURL, c.client.BaseURL)
	})

	t.Run("injected client keeps its own timeout", func(t *testing.T) {
		injected := resty.New().SetTimeout(5 * time.Second)
		c := NewHTTPClientWithRestyClient(baseURL, injected)
		assert.Equal(t, 5*time.Second, c.client.GetClient().Timeout)
		assert.Equal(t, baseURL, c.client.BaseURL)
	})
}

// opSigServer serves an operator signature per requested orderId and records the orderIds of
// every request it receives.
type opSigServer struct {
	*httptest.Server

	mu       sync.Mutex
	requests [][]string
}

func newOpSigServer(t *testing.T, expiresIn time.Duration) *opSigServer {
	t.Helper()

	srv := &opSigServer{}
	srv.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orderIds := r.URL.Query()["orderIds"]
		srv.mu.Lock()
		srv.requests = append(srv.requests, orderIds)
		srv.mu.Unlock()

		expiredAt := time.Now().Add(expiresIn).Unix()
		orders := make([]string, 0, len(orderIds))
		for _, id := range orderIds {
			orders = append(orders, fmt.Sprintf(
				`{"id":%s,"chainId":"1","operatorSignature":"0xsig%s","operatorSignatureExpiredAt":%d}`,
				id, id, expiredAt))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"code":0,"data":{"orders":[%s]}}`, strings.Join(orders, ","))
	}))
	t.Cleanup(srv.Close)

	return srv
}

func (s *opSigServer) recorded() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requests)
}

// TestGetOpSignatures_Cache covers the operator-signature cache: the backend re-signs only
// once the current expiry is nearly up, so a hit inside that window is byte-identical to a
// live call.
func TestGetOpSignatures_Cache(t *testing.T) {
	const (
		liveTTL      = time.Minute
		liveExpiry   = 90 * time.Second
		liveChainID  = ChainID(1)
		expiringSoon = defaultOpSignatureValidityMargin / 2
	)

	newClient := func(baseURL string, cacheTTL time.Duration) *httpClient {
		return NewHTTPClientWithConfig(&Config{
			LimitOrderHTTPUrl:   baseURL,
			OpSignatureCacheTTL: durationjson.Duration{Duration: cacheTTL},
		})
	}

	t.Run("a repeated orderId is served from cache", func(t *testing.T) {
		srv := newOpSigServer(t, liveExpiry)
		c := newClient(srv.URL, liveTTL)

		first, err := c.GetOpSignatures(t.Context(), liveChainID, []int64{1})
		require.NoError(t, err)
		second, err := c.GetOpSignatures(t.Context(), liveChainID, []int64{1})
		require.NoError(t, err)

		assert.Equal(t, [][]string{{"1"}}, srv.recorded())
		assert.Equal(t, first, second)
	})

	t.Run("only the uncached ids are fetched", func(t *testing.T) {
		srv := newOpSigServer(t, liveExpiry)
		c := newClient(srv.URL, liveTTL)

		_, err := c.GetOpSignatures(t.Context(), liveChainID, []int64{1, 2})
		require.NoError(t, err)
		sigs, err := c.GetOpSignatures(t.Context(), liveChainID, []int64{1, 2, 3})
		require.NoError(t, err)

		assert.Equal(t, [][]string{{"1", "2"}, {"3"}}, srv.recorded())

		signatureByID := make(map[int64]string, len(sigs))
		for _, sig := range sigs {
			signatureByID[sig.ID] = sig.OperatorSignature
		}
		assert.Equal(t, map[int64]string{1: "0xsig1", 2: "0xsig2", 3: "0xsig3"}, signatureByID)
	})

	t.Run("a signature expiring inside the validity margin is refetched", func(t *testing.T) {
		srv := newOpSigServer(t, expiringSoon)
		c := newClient(srv.URL, liveTTL)

		_, err := c.GetOpSignatures(t.Context(), liveChainID, []int64{1})
		require.NoError(t, err)
		_, err = c.GetOpSignatures(t.Context(), liveChainID, []int64{1})
		require.NoError(t, err)

		assert.Equal(t, [][]string{{"1"}, {"1"}}, srv.recorded())
	})

	t.Run("an entry past the cache TTL is refetched", func(t *testing.T) {
		srv := newOpSigServer(t, liveExpiry)
		c := newClient(srv.URL, time.Millisecond)

		_, err := c.GetOpSignatures(t.Context(), liveChainID, []int64{1})
		require.NoError(t, err)
		time.Sleep(20 * time.Millisecond)
		_, err = c.GetOpSignatures(t.Context(), liveChainID, []int64{1})
		require.NoError(t, err)

		assert.Equal(t, [][]string{{"1"}, {"1"}}, srv.recorded())
	})

	t.Run("zero TTL disables the cache", func(t *testing.T) {
		srv := newOpSigServer(t, liveExpiry)
		c := newClient(srv.URL, 0)

		for range 2 {
			_, err := c.GetOpSignatures(t.Context(), liveChainID, []int64{1})
			require.NoError(t, err)
		}

		assert.Equal(t, [][]string{{"1"}, {"1"}}, srv.recorded())
	})

	t.Run("concurrent callers share one cache", func(t *testing.T) {
		srv := newOpSigServer(t, liveExpiry)
		c := newClient(srv.URL, liveTTL)

		var wg sync.WaitGroup
		for i := range 32 {
			wg.Go(func() {
				sigs, err := c.GetOpSignatures(t.Context(), liveChainID, []int64{int64(i % 4)})
				assert.NoError(t, err)
				assert.Len(t, sigs, 1)
			})
		}
		wg.Wait()
	})
}
