package weighted

import (
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
)

type Extra struct {
	*shared.Extra
	NormalizedWeights []*uint256.Int `json:"normalizedWeights"`
}

type RpcResult struct {
	shared.RpcResult
	NormalizedWeights []*big.Int
}
