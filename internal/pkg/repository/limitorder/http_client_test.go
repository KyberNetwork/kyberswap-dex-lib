package limitorder

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"
)

func mockHandleListAllPairsSuccess(w http.ResponseWriter, r *http.Request) {
	jsonResponse := listAllPairsResult{
		Code:    0,
		Message: "Successfully",
		Data: &listAllPairsData{
			Pairs: []*valueobject.LimitOrderPair{
				{
					MakerAsset: "12",
					TakerAsset: "4333",
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(jsonResponse)
}

func mockHandleListOrdersSuccess(w http.ResponseWriter, r *http.Request) {
	jsonResponse := listOrdersResult{
		Code:    0,
		Message: "Successfully",
		Data: &listOrdersData{
			Orders: []*order{
				{
					ID:                   1,
					ChainID:              "1",
					Salt:                 "2222",
					Signature:            "222",
					MakerAsset:           "123",
					TakerAsset:           "2333",
					Maker:                "11",
					Receiver:             "22",
					AllowedSenders:       "22",
					MakingAmount:         "1000",
					TakingAmount:         "1000",
					FeeRecipient:         "777",
					FilledMakingAmount:   "1000",
					FilledTakingAmount:   "1000",
					MakerTokenFeePercent: "10",
					MakerAssetData:       "1",
					TakerAssetData:       "2",
					GetMakerAmount:       "3",
					GetTakerAmount:       "4",
					Predicate:            "5",
					Permit:               "6",
					Interaction:          "7",
					ExpiredAt:            111,
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(jsonResponse)
}

func Test_ListAllPairs(t *testing.T) {
	server, closeFunc := initServer()
	defer closeFunc()
	s := NewHTTPClient(server.URL)

	result, err := s.ListAllPairs(context.Background(), valueobject.ChainIDAurora)

	if err != nil {
		t.Errorf("TestGetConfigs failed, err: %v", err)
		return
	}

	want := []*valueobject.LimitOrderPair{
		{
			MakerAsset: "12",
			TakerAsset: "4333",
		},
	}

	assert.Equal(t, want, result)
}

func Test_ListOrders(t *testing.T) {
	server, closeFunc := initServer()
	defer closeFunc()
	s := NewHTTPClient(server.URL)

	result, err := s.ListOrders(context.Background(), service.ListOrdersFilter{
		ChainID:    valueobject.ChainIDEthereum,
		MakerAsset: "123",
		TakerAsset: "233",
	})

	if err != nil {
		t.Errorf("TestGetConfigs failed, err: %v", err)
		return
	}

	want := []*valueobject.Order{
		{
			ID:                   1,
			ChainID:              "1",
			Salt:                 "2222",
			Signature:            "222",
			MakerAsset:           "123",
			TakerAsset:           "2333",
			Maker:                "11",
			Receiver:             "22",
			AllowedSenders:       "22",
			MakingAmount:         big.NewInt(1000),
			TakingAmount:         big.NewInt(1000),
			FeeRecipient:         "777",
			FilledMakingAmount:   big.NewInt(1000),
			FilledTakingAmount:   big.NewInt(1000),
			MakerTokenFeePercent: 10,
			MakerAssetData:       "1",
			TakerAssetData:       "2",
			GetMakerAmount:       "3",
			GetTakerAmount:       "4",
			Predicate:            "5",
			Permit:               "6",
			Interaction:          "7",
			ExpiredAt:            111,
		},
	}

	assert.Equal(t, want, result)
}

func initServer() (*httptest.Server, func()) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				switch strings.TrimSpace(r.URL.Path) {
				case listAllPairsEndpoint:
					mockHandleListAllPairsSuccess(w, r)
				case listOrdersEndpoint:
					mockHandleListOrdersSuccess(w, r)
				default:
					http.NotFoundHandler().ServeHTTP(w, r)
				}
			},
		),
	)
	return server, server.Close
}
