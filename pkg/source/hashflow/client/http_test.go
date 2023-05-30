package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/stretchr/testify/assert"
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
			name: "it should return ErrListMarketMakersFailed when chainId is not valid",
			config: HTTPConfig{
				ChainID: valueobject.ChainIDEthereum,
				BaseURL: server.URL,
				APIKey:  "wrong APIKey",
				Source:  "kyber",
			},
			expectedError: ErrListMarketMakersFailed,
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
		rw.Write([]byte(`{"status":"fail","error":{"code":42,"message":"Missing source"}}`))

		return
	}

	if source != "kyber" {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte(fmt.Sprintf(`{"status":"fail","error":{"code":42,"message":"Invalid source: '%s'"}}`, source)))

		return
	}

	if authorizationHeader != "apiKey" {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(`{"status":"fail","error":{"code":72,"message":"Unauthorized access"}}`))

		return
	}

	networkID := queryParams.Get("networkId")

	if networkID == "999" {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(`{"status":"fail","error":{"code":42,"message":"Invalid networkId: 999"}}`))

		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(`{"marketMakers":["mm3_5","mm4","mm5","mm9","mm10_0","mm12_1","mm13","mm14_6","mm21"]}`))
}
