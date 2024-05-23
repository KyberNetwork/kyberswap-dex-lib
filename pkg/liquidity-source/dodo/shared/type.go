package shared

import (
	"math/big"

	"github.com/holiman/uint256"
)

type (
	Metadata struct {
		LastCreatedAtTimestamp *big.Int `json:"lastCreatedAtTimestamp"`
	}

	Token struct {
		Address  string `json:"id"`
		Name     string `json:"name"`
		Symbol   string `json:"symbol"`
		Decimals string `json:"decimals"`
	}

	SubgraphPool struct {
		ID                 string `json:"id"`
		BaseToken          Token  `json:"baseToken"`
		QuoteToken         Token  `json:"quoteToken"`
		BaseLpToken        Token  `json:"baseLpToken"`
		I                  string `json:"i"`
		K                  string `json:"k"`
		LpFeeRate          string `json:"lpFeeRate"`
		MtFeeRate          string `json:"mtFeeRate"`
		BaseReserve        string `json:"baseReserve"`
		QuoteReserve       string `json:"quoteReserve"`
		IsTradeAllowed     bool   `json:"isTradeAllowed"`
		Type               string `json:"type"`
		CreatedAtTimestamp string `json:"createdAtTimestamp"`
	}

	StaticExtra struct {
		PoolId           string   `json:"poolId"`
		LpToken          string   `json:"lpToken"`
		Type             string   `json:"type"`
		Tokens           []string `json:"tokens"`
		DodoV1SellHelper string   `json:"dodoV1SellHelper"`
	}
)

// V1 pool
type (
	V1TargetReserve struct {
		BaseTarget  *big.Int `json:"baseTarget"`
		QuoteTarget *big.Int `json:"quoteTarget"`
	}

	V1Extra struct {
		B           *uint256.Int `json:"B"`
		Q           *uint256.Int `json:"Q"`
		B0          *uint256.Int `json:"B0"`
		Q0          *uint256.Int `json:"Q0"`
		RStatus     int          `json:"rStatus"`
		OraclePrice *uint256.Int `json:"oraclePrice"`
		K           *uint256.Int `json:"k"`
		MtFeeRate   *uint256.Int `json:"mtFeeRate"`
		LpFeeRate   *uint256.Int `json:"lpFeeRate"`
		Swappable   bool         `json:"swappable"`
	}
)

// V2 pool
type (
	V2PMMState struct {
		I  *big.Int `json:"i"`
		K  *big.Int `json:"K"`
		B  *big.Int `json:"B"`
		Q  *big.Int `json:"Q"`
		B0 *big.Int `json:"B0"`
		Q0 *big.Int `json:"Q0"`
		R  *big.Int `json:"R"`
	}

	V2FeeRate struct {
		MtFeeRate *big.Int `json:"mtFeeRate"`
		LpFeeRate *big.Int `json:"lpFeeRate"`
	}

	V2Extra struct {
		I         *uint256.Int `json:"i"`
		K         *uint256.Int `json:"K"`
		B         *uint256.Int `json:"B"`
		Q         *uint256.Int `json:"Q"`
		B0        *uint256.Int `json:"B0"`
		Q0        *uint256.Int `json:"Q0"`
		R         *uint256.Int `json:"R"`
		MtFeeRate *uint256.Int `json:"mtFeeRate"`
		LpFeeRate *uint256.Int `json:"lpFeeRate"`
		Swappable bool         `json:"swappable"`
	}

	V2Meta struct {
		Type       string `json:"type"`
		BaseToken  string `json:"baseToken"`
		QuoteToken string `json:"quoteToken"`
	}

	V2Gas struct {
		SellBase  int64
		SellQuote int64
	}
)
