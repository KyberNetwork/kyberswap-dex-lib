package tickerdualpool

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/dualpool"
)

func TestQuoteRejectsImpossibleDistributionAndProtocolFees(t *testing.T) {
	base := dualpool.Extra{Live: true, Balance0: uint256.NewInt(1000000), Balance1: uint256.NewInt(1000000), SqrtPriceX96: new(uint256.Int).Lsh(uint256.NewInt(1), 96), Buckets: []dualpool.Bucket{{TickLower: -100, TickUpper: 100, WeightBps: 10000}}}
	for _, name := range []string{"too-many", "zero-weight", "out-of-range", "protocol-zero-for-one", "protocol-one-for-zero"} {
		t.Run(name, func(t *testing.T) {
			s := base
			s.Buckets = append([]dualpool.Bucket(nil), base.Buckets...)
			switch name {
			case "too-many":
				s.Buckets = make([]dualpool.Bucket, 9)
				for i := range s.Buckets {
					s.Buckets[i] = dualpool.Bucket{TickLower: -100, TickUpper: 100, WeightBps: 1000}
				}
				s.Buckets[0].WeightBps = 2000
			case "zero-weight":
				s.Buckets = append(s.Buckets, dualpool.Bucket{TickLower: -200, TickUpper: 200})
			case "out-of-range":
				s.Buckets[0].TickUpper = 887280
			case "protocol-zero-for-one":
				s.ProtocolFee = 1001
			case "protocol-one-for-zero":
				s.ProtocolFee = 1001 << 12
			}
			_, err := quoteTemporaryLiquidity(s, 10, true, uint256.NewInt(100))
			require.ErrorIs(t, err, ErrInvalidQuoteState)
		})
	}
}

func TestQuoteEnforcesPoolManagerLiquidityPerTick(t *testing.T) {
	state := dualpool.Extra{Live: true, Balance0: new(uint256.Int).Lsh(uint256.NewInt(1), 110), Balance1: new(uint256.Int).Lsh(uint256.NewInt(1), 110), SqrtPriceX96: new(uint256.Int).Lsh(uint256.NewInt(1), 96), Buckets: []dualpool.Bucket{{TickLower: -10, TickUpper: 10, WeightBps: 10000}}}
	_, err := quoteTemporaryLiquidity(state, 10, true, uint256.NewInt(100))
	require.ErrorIs(t, err, ErrInvalidQuoteState)
}
