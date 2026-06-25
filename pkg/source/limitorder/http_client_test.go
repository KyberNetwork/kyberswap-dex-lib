package limitorder

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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

		c := NewHTTPClientWithRestyClient(srv.URL, resty.New())
		_, err := c.GetOpSignatures(t.Context(), 1, []int64{123})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrGetOpSignaturesFailed), "got: %v", err)
	})

	t.Run("HTTP 4xx response wraps ErrGetOpSignaturesFailed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer srv.Close()

		c := NewHTTPClientWithRestyClient(srv.URL, resty.New())
		_, err := c.GetOpSignatures(t.Context(), 1, []int64{123})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrGetOpSignaturesFailed), "got: %v", err)
	})

	t.Run("transport/network error wraps ErrGetOpSignaturesFailed", func(t *testing.T) {
		c := NewHTTPClientWithRestyClient("http://127.0.0.1:1", resty.New())
		_, err := c.GetOpSignatures(t.Context(), 1, []int64{123})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrGetOpSignaturesFailed), "got: %v", err)
	})
}
