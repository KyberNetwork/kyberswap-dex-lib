package classical

import (
	"github.com/holiman/uint256"
)

type Meta struct {
	Type             string `json:"type"`
	DodoV1SellHelper string `json:"dodoV1SellHelper"`
	BaseToken        string `json:"baseToken"`
	QuoteToken       string `json:"quoteToken"`
}

type Gas struct {
	SellBase  int64
	SellQuote int64
	BuyBase   int64
}

type Storage struct {
	B              *uint256.Int // DODO._BASE_BALANCE_()
	Q              *uint256.Int // DODO._QUOTE_BALANCE_()
	B0             *uint256.Int // DODO._TARGET_BASE_TOKEN_AMOUNT_()
	Q0             *uint256.Int // DODO._TARGET_QUOTE_TOKEN_AMOUNT_()
	RStatus        int          // DODO._R_STATUS_()
	OraclePrice    *uint256.Int // DODO.getOraclePrice()
	K              *uint256.Int // DODO._K_()
	MtFeeRate      *uint256.Int // DODO._MT_FEE_RATE_()
	LpFeeRate      *uint256.Int // DODO._LP_FEE_RATE_()
	TradeAllowed   bool         // DODO._TRADE_ALLOWED_()
	SellingAllowed bool         // DODO._SELLING_ALLOWED_()
	BuyingAllowed  bool         // DODO._BUYING_ALLOWED_()
}
