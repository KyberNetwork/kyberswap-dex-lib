package setting

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func mockHandleSuccess(w http.ResponseWriter, r *http.Request) {
	jsonResponse := ConfigResponse{
		Code:    0,
		Message: "Successfully",
		Data: ConfigResponseData{
			Hash: "xyz",
			Config: ConfigResponseDataConfig{
				EnabledDexes: []valueobject.Dex{"uniswap", "uniswapv3", "dmm"},
				WhitelistedTokens: []valueobject.WhitelistedToken{
					{
						Address:  "address1",
						Name:     "name1",
						Symbol:   "symbol1",
						Decimals: 18,
						CgkId:    "cgkId1",
					},
				},
				BlacklistedPools: []string{"0x00"},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(jsonResponse)
}

func TestGetConfigs(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				switch strings.TrimSpace(r.URL.Path) {
				case "/api/v1/configurations":
					mockHandleSuccess(w, r)
				default:
					http.NotFoundHandler().ServeHTTP(w, r)
				}
			},
		),
	)
	defer server.Close()

	s := NewRestRepository(server.URL + "/api/v1/configurations")

	result, err := s.GetConfigs(context.Background(), "aggregator-1", "")

	if err != nil {
		t.Errorf("TestGetConfigs failed, err: %v", err)
		return
	}

	want := valueobject.RemoteConfig{
		Hash:         "xyz",
		EnabledDexes: []valueobject.Dex{"uniswap", "uniswapv3", "dmm"},
		WhitelistedTokens: []valueobject.WhitelistedToken{
			{
				Address:  "address1",
				Name:     "name1",
				Symbol:   "symbol1",
				Decimals: 18,
				CgkId:    "cgkId1",
			},
		},
		BlacklistedPools: []string{"0x00"},
	}

	assert.Equal(t, want, result)
}
