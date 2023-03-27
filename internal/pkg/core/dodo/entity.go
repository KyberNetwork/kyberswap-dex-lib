package dodo

import (
	"math/big"
)

type Gas struct {
	SellBaseV1 int64
	BuyBaseV1  int64
	SellBaseV2 int64
	BuyBaseV2  int64
}

type PoolState struct {
	B           *big.Float // DODO._BASE_BALANCE_() / 10^baseDecimals
	Q           *big.Float // DODO._QUOTE_BALANCE_() / 10^quoteDecimals
	B0          *big.Float // DODO._TARGET_BASE_TOKEN_AMOUNT_() / 10^baseDecimals
	Q0          *big.Float // DODO._TARGET_QUOTE_TOKEN_AMOUNT_() / 10^quoteDecimals
	RStatus     int        // DODO._R_STATUS_()
	OraclePrice *big.Float // DODO.getOraclePrice() / 10^(18-baseDecimals+quoteDecimals)
	k           *big.Float // DODO._K_()/10^18
	mtFeeRate   *big.Float // DODO._MT_FEE_RATE_()/10^18
	lpFeeRate   *big.Float // DODO._LP_FEE_RATE_()/10^18
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

type StaticExtra struct {
	PoolId           string   `json:"poolId"`
	LpToken          string   `json:"lpToken"`
	Type             string   `json:"type"`
	Tokens           []string `json:"tokens"`
	DodoV1SellHelper string   `json:"dodoV1SellHelper"`
}

type Meta struct {
	Type             string `json:"type"`
	DodoV1SellHelper string `json:"dodoV1SellHelper"`
	BaseToken        string `json:"baseToken"`
	QuoteToken       string `json:"quoteToken"`
}
