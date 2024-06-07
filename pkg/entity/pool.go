package entity

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolReserves []string

func (r PoolReserves) Encode() string {
	return strings.Join(r, ":")
}

type PoolToken struct {
	Address   string `json:"address,omitempty"`
	Name      string `json:"name,omitempty"`
	Symbol    string `json:"symbol,omitempty"`
	Decimals  uint8  `json:"decimals,omitempty"`
	Weight    uint   `json:"weight,omitempty"`
	Swappable bool   `json:"swappable,omitempty"`
}

type PoolTokens []*PoolToken

func ClonePoolTokens(poolTokens []*PoolToken) []*PoolToken {
	var result = make([]*PoolToken, len(poolTokens))
	for i, poolToken := range poolTokens {
		clonePoolToken := &PoolToken{
			Address:   poolToken.Address,
			Name:      poolToken.Name,
			Symbol:    poolToken.Symbol,
			Decimals:  poolToken.Decimals,
			Weight:    poolToken.Weight,
			Swappable: poolToken.Swappable,
		}
		result[i] = clonePoolToken
	}
	return result
}

type Pool struct {
	Address      string       `json:"address,omitempty"`
	ReserveUsd   float64      `json:"reserveUsd,omitempty"`
	AmplifiedTvl float64      `json:"amplifiedTvl,omitempty"`
	SwapFee      float64      `json:"swapFee,omitempty"`
	Exchange     string       `json:"exchange,omitempty"`
	Type         string       `json:"type,omitempty"`
	Timestamp    int64        `json:"timestamp,omitempty"`
	Reserves     PoolReserves `json:"reserves,omitempty"`
	Tokens       []*PoolToken `json:"tokens,omitempty"`
	Extra        string       `json:"extra,omitempty"`
	StaticExtra  string       `json:"staticExtra,omitempty"`
	TotalSupply  string       `json:"totalSupply,omitempty"`
	BlockNumber  uint64       `json:"blockNumber,omitempty"`
}

func (p Pool) IsZero() bool { return len(p.Address) == 0 && len(p.Tokens) == 0 }

func (p Pool) GetTotalSupply() float64 {
	totalSupplyBF, _ := new(big.Float).SetString(p.TotalSupply)
	totalSupply, _ := new(big.Float).Quo(totalSupplyBF, bignumber.TenPowDecimals(18)).Float64()

	return totalSupply
}

// GetLpToken returns the LpToken of the pool
// If there is a LpToken in the StaticExtra, we use it. If not, we get the pool's address
func (p Pool) GetLpToken() string {

	var staticExtra = struct {
		LpToken string `json:"lpToken"`
	}{}

	_ = json.Unmarshal([]byte(p.StaticExtra), &staticExtra)

	if len(staticExtra.LpToken) > 0 {
		return strings.ToLower(staticExtra.LpToken)
	}

	return p.Address
}

// HasReserves check if a pool has correct reserves or not
// if there is no reserve in pool, or reserve is empty string, or reserve = "0", this function returns false
// if pool has equals or more than 2 tokens have reserve, this function returns true
func (p Pool) HasReserves() bool {
	if (len(p.Reserves)) == 0 {
		return false
	}

	zeroReserveCount := 0
	for _, reserve := range p.Reserves {
		if len(reserve) == 0 || reserve == "0" {
			zeroReserveCount += 1
		}
	}

	return len(p.Reserves)-zeroReserveCount >= 2
}

func (p Pool) HasReserve(reserve string) bool {
	if len(reserve) == 0 || reserve == "0" {
		return false
	}

	return true
}

// HasAmplifiedTvl check if the pool has amplifiedTvl or not
func (p Pool) HasAmplifiedTvl() bool {
	return p.AmplifiedTvl > 0
}

func (p *Pool) Clear() {
	p.Type = ""
	if p.Reserves != nil {
		// Keep allocated memory
		for i := range p.Reserves {
			p.Reserves[i] = "0"
		}
		p.Reserves = p.Reserves[:0]
	}
	p.Address = ""
	if p.Tokens != nil {
		// Keep allocated memory
		for i := range p.Tokens {
			p.Tokens[i].Weight = 0
			p.Tokens[i].Name = ""
			p.Tokens[i].Swappable = false
			p.Tokens[i].Address = ""
			p.Tokens[i].Symbol = ""
			p.Tokens[i].Decimals = 0
		}
		p.Tokens = p.Tokens[:0]
	}
	p.Extra = ""
	p.StaticExtra = ""
	p.Exchange = ""
	p.ReserveUsd = 0
	p.Timestamp = 0
	p.SwapFee = 0
	p.AmplifiedTvl = 0
	p.TotalSupply = ""
}
