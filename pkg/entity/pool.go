package entity

import (
	"math/big"
	"slices"
	"strings"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolReserves []string

func (r PoolReserves) Encode() string {
	return strings.Join(r, ":")
}

type PoolToken struct {
	Address   string `json:"address,omitempty"`
	Symbol    string `json:"symbol,omitempty"`
	Decimals  uint8  `json:"decimals,omitempty"`
	Swappable bool   `json:"swappable,omitempty"`
}

type PoolTokens []*PoolToken

func ClonePoolTokens(poolTokens []*PoolToken) []*PoolToken {
	var result = make([]*PoolToken, len(poolTokens))
	for i, poolToken := range poolTokens {
		clonePoolToken := &PoolToken{
			Address:   poolToken.Address,
			Symbol:    poolToken.Symbol,
			Decimals:  poolToken.Decimals,
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

func (p *Pool) IsZero() bool { return len(p.Address) == 0 && len(p.Tokens) == 0 }

func (p *Pool) GetTotalSupply() float64 {
	totalSupplyBF, _ := new(big.Float).SetString(p.TotalSupply)
	totalSupply, _ := totalSupplyBF.Quo(totalSupplyBF, bignumber.TenPowDecimals(18)).Float64()
	return totalSupply
}

// GetLpToken returns the LpToken of the pool
// If there is a LpToken in the StaticExtra, we use it. If not, we get the pool's address
func (p *Pool) GetLpToken() string {
	var staticExtra = struct {
		LpToken string `json:"lpToken"`
	}{}
	_ = json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	if len(staticExtra.LpToken) > 0 {
		return strings.ToLower(staticExtra.LpToken)
	}

	return p.Address
}

// HasReserves check if a pool has some reserves or not.
// Returns false if there is no reserve in pool, or all reserves are empty string or "0". True otherwise.
func (p *Pool) HasReserves() bool {
	return slices.ContainsFunc(p.Reserves, p.HasReserve)
}

func (p *Pool) HasReserve(reserve string) bool {
	return len(reserve) > 0 && reserve != "0"
}

// HasAmplifiedTvl check if the pool has amplifiedTvl or not
func (p *Pool) HasAmplifiedTvl() bool {
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
			p.Tokens[i].Address = ""
			p.Tokens[i].Symbol = ""
			p.Tokens[i].Decimals = 0
			p.Tokens[i].Swappable = false
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
	p.BlockNumber = 0
}
