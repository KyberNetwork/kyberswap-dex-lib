package base

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func validateExtra(extra *shared.Extra) error {
	if extra == nil ||
		len(extra.BalancesLiveScaled18) == 0 ||
		len(extra.DecimalScalingFactors) == 0 ||
		len(extra.TokenRates) == 0 {
		return shared.ErrInvalidExtra
	}

	return nil
}

func GetRouterAddress(chainID valueobject.ChainID) (common.Address, bool) {
	v, ok := BalancerV3BatchRouter[chainID]
	return v, ok
}
