package dodo

import "math/big"

type Metadata map[string]PoolTypeMetadata

type PoolTypeMetadata struct {
	LastCreatedAtTimestamp *big.Int `json:"lastCreatedAtTimestamp"`
}

type Token struct {
	Address  string `json:"id"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals string `json:"decimals"`
}

type SubgraphPool struct {
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

type StaticExtra struct {
	PoolId           string   `json:"poolId"`
	LpToken          string   `json:"lpToken"`
	Type             string   `json:"type"`
	Tokens           []string `json:"tokens"`
	DodoV1SellHelper string   `json:"dodoV1SellHelper"`
}

type Extra struct {
	I              *big.Int   `json:"i"`
	K              *big.Int   `json:"k"`
	RStatus        int        `json:"rStatus"`
	MtFeeRate      *big.Float `json:"mtFeeRate"`
	LpFeeRate      *big.Float `json:"lpFeeRate"`
	Swappable      bool       `json:"swappable"`
	Reserves       []*big.Int `json:"reserves"`
	TargetReserves []*big.Int `json:"targetReserves"`
}

type TargetReserve struct {
	BaseTarget  *big.Int `json:"baseTarget"`
	QuoteTarget *big.Int `json:"quoteTarget"`
}

// type for DodoV2

type PoolState struct {
	I  *big.Int `json:"i"`
	K  *big.Int `json:"K"`
	B  *big.Int `json:"B"`
	Q  *big.Int `json:"Q"`
	B0 *big.Int `json:"B0"`
	Q0 *big.Int `json:"Q0"`
	R  *big.Int `json:"R"`
}

type FeeRate struct {
	MtFeeRate *big.Int `json:"mtFeeRate"`
	LpFeeRate *big.Int `json:"lpFeeRate"`
}
