package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestHttpClient_ListMarketMakers(t *testing.T) {
	server := initServer()
	defer server.Close()

	testCases := []struct {
		name                 string
		config               HTTPConfig
		expectedMarketMakers []string
		expectedError        error
	}{
		{
			name: "it should return ErrListMarketMakersFailed when apiKey is not valid",
			config: HTTPConfig{
				ChainID: valueobject.ChainIDEthereum,
				BaseURL: server.URL,
				APIKey:  "invalid-apiKey",
				Source:  "kyber",
			},
			expectedError: ErrListMarketMakersFailed,
		},
		{
			name: "it should return ErrListMarketMakersFailed when source is not valid",
			config: HTTPConfig{
				ChainID: valueobject.ChainIDEthereum,
				BaseURL: server.URL,
				APIKey:  "apiKey",
				Source:  "invalid-source",
			},
			expectedError: ErrListMarketMakersFailed,
		},
		{
			name: "it should return ErrListMarketMakersFailed when chainId is not valid",
			config: HTTPConfig{
				ChainID: 10,
				BaseURL: server.URL,
				APIKey:  "apiKey",
				Source:  "kyber",
			},
			expectedError: ErrListMarketMakersFailed,
		},
		{
			name: "it should return market makers when request is valid",
			config: HTTPConfig{
				ChainID: valueobject.ChainIDEthereum,
				BaseURL: server.URL,
				APIKey:  "apiKey",
				Source:  "kyber",
			},
			expectedMarketMakers: []string{"mm3_5", "mm4", "mm5", "mm9", "mm10_0", "mm12_1", "mm13", "mm14_6", "mm21"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			client := NewHTTPClient(&tc.config)

			marketMakers, err := client.ListMarketMakers(ctx)

			assert.Equal(t, tc.expectedMarketMakers, marketMakers)
			assert.ErrorIs(t, err, tc.expectedError)
		})
	}
}

func initServer() *httptest.Server {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case listMarketMakersPath:
					mockListMarketMakersHandler(w, r)
				default:
					http.NotFoundHandler().ServeHTTP(w, r)
				}
			}),
	)

	return server
}

func mockListMarketMakersHandler(rw http.ResponseWriter, r *http.Request) {
	authorizationHeader := r.Header.Get("Authorization")

	if len(authorizationHeader) == 0 {
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	queryParams := r.URL.Query()

	source := queryParams.Get("source")

	if len(source) == 0 {
		rw.WriteHeader(http.StatusBadRequest)
		// nolint:errcheck
		rw.Write([]byte(`{"status":"fail","error":{"code":42,"message":"Missing source"}}`))

		return
	}

	if source != "kyber" {
		rw.WriteHeader(http.StatusBadRequest)
		// nolint:errcheck
		rw.Write([]byte(fmt.Sprintf(`{"status":"fail","error":{"code":42,"message":"Invalid source: '%s'"}}`, source)))

		return
	}

	if authorizationHeader != "apiKey" {
		rw.WriteHeader(http.StatusUnauthorized)
		// nolint:errcheck
		rw.Write([]byte(`{"status":"fail","error":{"code":72,"message":"Unauthorized access"}}`))

		return
	}

	networkID := queryParams.Get("networkId")
	if networkID != strconv.FormatUint(uint64(valueobject.ChainIDEthereum), 10) {
		rw.WriteHeader(http.StatusUnauthorized)
		// nolint:errcheck
		rw.Write([]byte(fmt.Sprintf(`{"status":"fail","error":{"code":42,"message":"Invalid networkId: %s"}}`, networkID)))

		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	// nolint:errcheck
	rw.Write([]byte(`{"marketMakers":["mm3_5","mm4","mm5","mm9","mm10_0","mm12_1","mm13","mm14_6","mm21"]}`))
}
