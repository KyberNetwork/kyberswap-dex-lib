package woofiv2

import "math/big"

type WooFiV2State struct {
	QuoteToken    string                `json:"quoteToken"`
	UnclaimedFee  *big.Int              `json:"unclaimedFee"`
	TokenInfos    map[string]*TokenInfo `json:"tokenInfos"`
	Timestamp     *big.Int              `json:"timestamp"`
	StaleDuration *big.Int              `json:"staleDuration"`
	Bound         *big.Int              `json:"bound"`
}

type wooFiV2SwapInfo struct {
	unclaimedFee *big.Int
	tokenInfos   map[string]*TokenInfo
}

type Extra struct {
	QuoteToken    string                `json:"quoteToken"`
	UnclaimedFee  *big.Int              `json:"unclaimedFee"`
	Wooracle      string                `json:"wooracle"`
	TokenInfos    map[string]*TokenInfo `json:"tokenInfos"`
	Timestamp     *big.Int              `json:"timestamp"`
	StaleDuration *big.Int              `json:"staleDuration"`
	Bound         *big.Int              `json:"bound"`
}

type TokenInfo struct {
	Reserve  *big.Int     `json:"reserve"`
	FeeRate  *big.Int     `json:"feeRate"`
	Decimals uint8        `json:"decimals"`
	State    *OracleState `json:"state"`
}

type OracleState struct {
	Price        *big.Int `json:"price"`
	Spread       *big.Int `json:"spread"`
	Coeff        *big.Int `json:"coeff"`
	WoFeasible   bool     `json:"woFeasible"`
	Decimals     uint8    `json:"decimals"`
	CloPrice     *big.Int `json:"cloPrice"`
	CloPreferred bool     `json:"cloPreferred"`
}

type DecimalInfo struct {
	PriceDec *big.Int `json:"priceDec"`
	QuoteDec *big.Int `json:"quoteDec"`
	BaseDec  *big.Int `json:"baseDec"`
}

type Gas struct {
	Swap int64
}
