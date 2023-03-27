package httputils

import (
	"net/http"

	"context"

	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

type Token struct {
	Address  string
	Name     string
	Symbol   string
	Decimals int
	CkgID    string `json:"cgkId"`
}

type TokenInfo struct {
	Address  string
	Name     string
	Symbol   string
	Decimals int
	Price    float64
	Type     string
	CkgID    string   `json:"cgkId"`
	Tokens   []*Token `json:"tokens"`
}

type PairInfo struct {
	Address     string
	Token0      string
	Token1      string
	Reserve0    float64
	Reserve1    float64
	Weight0     int
	Weight1     int
	Decimals0   int
	Decimals1   int
	ReserveUsd  float64
	TotalSupply float64
	SwapFee     float64
	Exchange    string
	Timestamp   int
}

func GetTokenInfo(ctx context.Context, api, address string) (*TokenInfo, error) {
	api += "/api/token/token-info"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
	if err != nil {
		logger.Errorf("failed to prepare client request, err: %v", err)
		return nil, err
	}
	q := req.URL.Query()
	q.Add("addresses", address)
	req.URL.RawQuery = q.Encode()

	info := make(map[string]*TokenInfo)
	if err := Process(ctx, req, &info); err != nil {
		logger.Errorf("failed to call price api, err: %v", err)
		return nil, err
	}

	return info[address], nil
}

func GetPairInfo(ctx context.Context, api, address string) (*PairInfo, error) {
	api += "/api/pair/pair-info"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
	if err != nil {
		logger.Errorf("failed to prepare client request, err: %v", err)
		return nil, err
	}
	q := req.URL.Query()
	q.Add("addresses", address)
	req.URL.RawQuery = q.Encode()

	info := make(map[string]*PairInfo)
	if err := Process(ctx, req, &info); err != nil {
		return nil, err
	}

	return info[address], nil
}

func GetTokenPrice(ctx context.Context, api, address string) (float64, error) {
	api += "/api/price/token-price"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
	if err != nil {
		logger.Errorf("failed to prepare client request, err: %v", err)
		return 0, err
	}
	q := req.URL.Query()
	q.Add("addresses", address)
	req.URL.RawQuery = q.Encode()

	info := make(map[string]float64)
	if err := Process(ctx, req, &info); err != nil {
		return 0, err
	}

	return info[address], nil
}
