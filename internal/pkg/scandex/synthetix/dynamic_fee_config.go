package synthetix

import "math/big"

type DynamicFeeConfig struct {
	Threshold   *big.Int `json:"threshold"`
	WeightDecay *big.Int `json:"weightDecay"`
	Rounds      *big.Int `json:"rounds"`
	MaxFee      *big.Int `json:"maxFee"`
}

func NewDynamicFeeConfig() *DynamicFeeConfig {
	return &DynamicFeeConfig{}
}
