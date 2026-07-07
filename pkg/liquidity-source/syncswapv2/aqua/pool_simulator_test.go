package syncswapv2aqua

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	// test data from https://explorer.zksync.io/address/0x50e00Ac0B02fEdB1b8044A565B86311425b1355f
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"A", 1000000000000000000, "B", 18660541676},
	}

	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"8466391136317679557", "193408158540"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    "{\"swapFee0To1Min\":800,\"swapFee0To1Max\":1000,\"swapFee0To1Gamma\":230000000000000,\"swapFee1To0Min\":800,\"swapFee1To0Max\":1000,\"swapFee1To0Gamma\":230000000000000,\"token0PrecisionMultiplier\":1,\"token1PrecisionMultiplier\":1000000000,\"vaultAddress\":\"0x621425a1Ef6abE91058E9712575dcc4258F8d091\",\"priceScale\":54451990779514461,\"a\":4000000,\"d\":18973521177677971086,\"gamma\":1450000000000000,\"futureTime\":1709616182,\"lastPrices\":49576568810461066,\"priceOracle\":49663786937733135,\"lastPricesTimestamp\":1716279641,\"lpSupply\":37758794556622160853,\"xcpProfit\":1152938294335819254,\"virtualPrice\":1076695600534779561,\"allowedExtraProfit\":2000000000000,\"adjustmentStep\":146000000000000,\"maHalfTime\":600}",
	})
	require.Nil(t, err)

	assert.Equal(t, 0, len(p.CanSwapTo("LP")))
	assert.Equal(t, []string{"B"}, p.CanSwapTo("A"))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)},
					TokenOut:      tc.out,
					Limit:         nil,
				})
			})
			require.Nil(t, err)
			assert.Equal(t, big.NewInt(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
		})
	}
}
