package composable

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type bptSimulator struct {
	poolpkg.Pool

	swapFeePercentage                   *uint256.Int
	scalingFactors                      []*uint256.Int
	bptIndex                            *uint256.Int
	amp                                 *uint256.Int
	bptTotalSupply                      *uint256.Int
	protocolFeePercentageCacheSwapType  *uint256.Int
	protocolFeePercentageCacheYieldType *uint256.Int

	lastJoinExit                     LastJoinExitData
	rateProviders                    []string
	tokensExemptFromYieldProtocolFee []bool
	tokenRateCaches                  []TokenRateCache
}

func (s *bptSimulator) swap(
	amountIn *uint256.Int,
	balances []*uint256.Int,
	indexIn int,
	indexOut int,
) (*uint256.Int, *poolpkg.TokenAmount, error) {
	return nil, nil, nil
}

func (s *bptSimulator) _getVirtualSupply(bptBalance *uint256.Int) (*uint256.Int, error) {
	cir, err := math.FixedPoint.Sub(s.bptTotalSupply, bptBalance)
	if err != nil {
		return nil, err
	}
	return cir, nil
}
