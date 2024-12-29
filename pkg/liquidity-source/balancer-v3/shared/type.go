package shared

import "github.com/holiman/uint256"

type PoolInfo struct {
	Name       string `json:"-"`
	Version    int    `json:"version"`
	Deployment string `json:"-"`
}

type VaultSwapParams struct {
	IsExactIn                  bool
	IndexIn                    int
	IndexOut                   int
	AmountGiven                *uint256.Int
	DecimalScalingFactor       *uint256.Int
	TokenRate                  *uint256.Int
	AmplificationParameter     *uint256.Int
	SwapFeePercentage          *uint256.Int
	AggregateSwapFeePercentage *uint256.Int
	BalancesLiveScaled18       []*uint256.Int
}
