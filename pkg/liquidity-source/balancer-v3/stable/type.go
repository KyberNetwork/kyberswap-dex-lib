package stable

import (
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
)

type Extra struct {
	*shared.Extra
	SurgePercentages       `json:"surge"`
	AmplificationParameter *uint256.Int `json:"ampParam,omitempty"`
}

type SurgePercentages struct {
	MaxSurgeFeePercentage    *uint256.Int `json:"max,omitempty"`
	SurgeThresholdPercentage *uint256.Int `json:"thres,omitempty"`
}

type AmplificationParameterRpc struct {
	Value      *big.Int
	IsUpdating bool
	Precision  *big.Int
}

type SurgePercentagesRpc struct {
	MaxSurgeFeePercentage    *big.Int
	SurgeThresholdPercentage *big.Int
}

type RpcResult struct {
	shared.RpcResult
	SurgePercentagesRpc
	AmplificationParameterRpc
}
