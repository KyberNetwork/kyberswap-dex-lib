package dto

import "github.com/KyberNetwork/router-service/internal/pkg/entity"

type (
	GetPoolsResult struct {
		Pools []*GetPoolsResultPool `json:"pools"`
	}

	GetPoolsResultPool struct {
		ReserveUsd   float64                    `json:"reserveUsd,omitempty"`
		AmplifiedTvl float64                    `json:"amplifiedTvl,omitempty"`
		SwapFee      float64                    `json:"swapFee,omitempty"`
		Exchange     string                     `json:"exchange,omitempty"`
		Type         string                     `json:"type,omitempty"`
		Timestamp    int64                      `json:"timestamp,omitempty"`
		Reserves     []string                   `json:"reserves,omitempty"`
		Tokens       []*GetPoolsResultPoolToken `json:"tokens,omitempty"`
		Extra        string                     `json:"extra,omitempty"`
		StaticExtra  string                     `json:"staticExtra,omitempty"`
		TotalSupply  string                     `json:"totalSupply,omitempty"`
	}

	GetPoolsResultPoolToken struct {
		Address   string `json:"address"`
		Name      string `json:"name"`
		Symbol    string `json:"symbol"`
		Decimals  uint8  `json:"decimals"`
		Weight    uint   `json:"weight"`
		Swappable bool   `json:"swappable"`
	}
)

func NewGetPoolsResult(pools []entity.Pool) *GetPoolsResult {
	resultPools := make([]*GetPoolsResultPool, 0, len(pools))
	for _, pool := range pools {
		resultPools = append(resultPools, toGetPoolsResultPool(pool))
	}

	return &GetPoolsResult{
		Pools: resultPools,
	}
}

func toGetPoolsResultPool(pool entity.Pool) *GetPoolsResultPool {
	reserves := make([]string, 0, len(pool.Reserves))
	for _, reserve := range pool.Reserves {
		reserves = append(reserves, reserve)
	}

	return &GetPoolsResultPool{
		ReserveUsd:   pool.ReserveUsd,
		AmplifiedTvl: pool.AmplifiedTvl,
		SwapFee:      pool.SwapFee,
		Exchange:     pool.Exchange,
		Type:         pool.Type,
		Timestamp:    pool.Timestamp,
		Reserves:     reserves,
		Tokens:       toGetPoolsResultPoolTokens(pool.Tokens),
		Extra:        pool.Extra,
		StaticExtra:  pool.StaticExtra,
		TotalSupply:  pool.TotalSupply,
	}
}

func toGetPoolsResultPoolTokens(tokens []*entity.PoolToken) []*GetPoolsResultPoolToken {
	respTokens := make([]*GetPoolsResultPoolToken, 0, len(tokens))

	for _, token := range tokens {
		if token == nil {
			respTokens = append(respTokens, nil)
			continue
		}

		respTokens = append(respTokens, &GetPoolsResultPoolToken{
			Address:   token.Address,
			Name:      token.Name,
			Symbol:    token.Symbol,
			Decimals:  token.Decimals,
			Weight:    token.Weight,
			Swappable: token.Swappable,
		})
	}

	return respTokens
}
