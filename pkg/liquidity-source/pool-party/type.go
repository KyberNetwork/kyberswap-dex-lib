package poolparty

import (
	"math/big"
)

type MetaInfo struct {
	Exchange string `mapstructure:"exchange"`
}

type Metadata struct {
	LastCreatedAtTimestamp int      `json:"lastCreatedAtTimestamp"`
	LastPoolIds            []string `json:"lastPoolIds"`
}

type Extra struct {
	PoolStatus            string   `json:"poolStatus"`
	IsVisible             bool     `json:"isVisible"`
	BoostPriceBps         int      `json:"boostPriceBps"`
	RateToETH             *big.Int `json:"rateToETH"` // Rate between 1 src token to ETH
	PublicAmountAvailable *big.Int `json:"publicAmountAvailable"`
	Exchange              string   `json:"exchange"`
}

type SubgraphPool struct {
	ID                    string `json:"id"`
	TokenAddress          string `json:"tokenAddress"`
	TokenSymbol           string `json:"tokenSymbol"`
	TokenDecimals         int    `json:"tokenDecimals"`
	IsVisible             bool   `json:"isVisible"`
	PoolStatus            string `json:"poolStatus"`
	PublicAmountAvailable string `json:"publicAmountAvailable"`
	CreatedAtTimestamp    string `json:"createdAtTimestamp"`
}
