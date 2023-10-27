package levelfinance

import "math/big"

type PoolState struct {
	TokenInfos       map[string]*TokenInfo `json:"tokenInfos"`
	TotalWeight      *big.Int              `json:"totalWeight"`
	VirtualPoolValue *big.Int              `json:"virtualPoolValue"`

	// fee
	StableCoinBaseSwapFee   *big.Int `json:"stableCoinBaseSwapFee"`
	StableCoinTaxBasisPoint *big.Int `json:"stableCoinTaxBasisPoint"`
	BaseSwapFee             *big.Int `json:"baseSwapFee"`
	TaxBasisPoint           *big.Int `json:"taxBasisPoint"`
	DaoFee                  *big.Int `json:"daoFee"`
}

type Extra struct {
	Oracle           string                `json:"oracle"`
	TotalWeight      *big.Int              `json:"totalWeight"`
	VirtualPoolValue *big.Int              `json:"virtualPoolValue"`
	TokenInfos       map[string]*TokenInfo `json:"tokenInfos"`

	// fee
	StableCoinBaseSwapFee   *big.Int `json:"stableCoinBaseSwapFee"`
	StableCoinTaxBasisPoint *big.Int `json:"stableCoinTaxBasisPoint"`
	BaseSwapFee             *big.Int `json:"baseSwapFee"`
	TaxBasisPoint           *big.Int `json:"taxBasisPoint"`
	DaoFee                  *big.Int `json:"daoFee"`
}

type TokenInfo struct {
	IsStableCoin bool `json:"isStableCoin"`

	TargetWeight *big.Int `json:"targetWeight"`

	TrancheAssets   map[string]*AssetInfo `json:"trancheAssets"`
	RiskFactor      map[string]*big.Int   `json:"riskFactor"`
	TotalRiskFactor *big.Int              `json:"totalRiskFactor"`

	// oracle.getPrice(_token, false)
	MinPrice *big.Int `json:"minPrice"`
	// oracle.getPrice(_token, true)
	MaxPrice *big.Int `json:"maxPrice"`
}

type AssetInfo struct {
	PoolAmount    *big.Int `json:"poolAmount"`
	ReserveAmount *big.Int `json:"reserveAmount"`
}

type Gas struct {
	Swap int64
}
