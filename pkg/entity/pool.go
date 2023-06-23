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
func (p Pool) HasReserves() bool {
	if (len(p.Reserves)) == 0 {
		return false
	}

	for _, reserve := range p.Reserves {
		if len(reserve) == 0 || reserve == "0" {
			return false
		}
	}

	return true
}

// HasAmplifiedTvl check if the pool has amplifiedTvl or not
func (p Pool) HasAmplifiedTvl() bool {
	return p.AmplifiedTvl > 0
}
