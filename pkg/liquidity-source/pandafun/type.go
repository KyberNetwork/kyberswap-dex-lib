package pandafun

import "math/big"

type Extra struct {
	MinTradeSize               *big.Int `json:"minTradeSize"`
	AmountInBuyRemainingTokens *big.Int `json:"amountInBuyRemainingTokens"`
	Liquidity                  *big.Int `json:"liquidity"`
	BuyFee                     *big.Int `json:"buyFee"`
	SellFee                    *big.Int `json:"sellFee"`
	SqrtPa                     *big.Int `json:"sqrtPa"`
	SqrtPb                     *big.Int `json:"sqrtPb"`
}

type PoolFees struct {
	BuyFee           uint16
	SellFee          uint16
	GraduationFee    uint16
	DeployerFeeShare uint16
}

type Metadata struct {
	Offset int `json:"offset"`
}
