package base

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"

func validateExtra(extra *shared.Extra) error {
	if extra == nil ||
		len(extra.BalancesLiveScaled18) == 0 ||
		len(extra.DecimalScalingFactors) == 0 ||
		len(extra.TokenRates) == 0 {
		return shared.ErrInvalidExtra
	}

	return nil
}
