package uniswapv3pt

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
)

type PoolTicksClient struct {
	baseURL    *url.URL
	httpClient *http.Client
}

func NewPoolTicksClient(baseURL string, httpClient *http.Client) (*PoolTicksClient, error) {
	if len(baseURL) == 0 {
		return nil, errors.New("baseURL is empty")
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid baseURL: %w", err)
	}

	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &PoolTicksClient{
		baseURL:    parsedURL,
		httpClient: httpClient,
	}, nil
}

type PoolTick struct {
	TickIdx        int      `json:"tickIdx"`
	LiquidityGross *big.Int `json:"liquidityGross"`
	LiquidityNet   *big.Int `json:"liquidityNet"`
}

func (c *PoolTicksClient) GetPoolTicks(poolAddress string) ([]PoolTick, error) {
	u := c.baseURL.JoinPath("pools").JoinPath(poolAddress)
	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var response struct {
		Success bool   `json:"success"`
		Reason  string `json:"reason"`
		Data    struct {
			Ticks []PoolTick `json:"ticks"`
		} `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	if !response.Success {
		return nil, errors.New(response.Reason)
	}

	return response.Data.Ticks, nil
}
