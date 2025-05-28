package two

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
	// test data from https://etherscan.io/address/0x95f3672a418230c5664b7154dfce0acfa7eed68d#readContract
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"A", 10, "B", 304},
		{"B", 1000, "A", 172},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange:    "",
		Type:        "",
		Reserves:    entity.PoolReserves{"2575977394749099472751", "1447320191806527553931"},
		Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:       "{\"A\":\"200000000\",\"D\":\"4344269418800893049364\",\"gamma\":\"100000000000000\",\"priceScale\":\"1250033866036595049\",\"lastPrices\":\"1241874208010789089\",\"priceOracle\":\"1199834141509881054\",\"feeGamma\":\"5000000000000000\",\"midFee\":\"10000000\",\"outFee\":\"90000000\",\"futureAGammaTime\":0,\"futureAGamma\":\"68056473384187692692674921486353742291200000000\",\"initialAGammaTime\":0,\"initialAGamma\":\"68056473384187692692674921486353742291200000000\",\"lastPricesTimestamp\":1686876995,\"lpSupply\":\"1894549993474267797965\",\"xcpProfit\":\"1034188512253919548\",\"virtualPrice\":\"1025462529694819838\",\"allowedExtraProfit\":\"10000000000\",\"adjustmentStep\":\"5500000000000\",\"maHalfTime\":\"600\"}",
		StaticExtra: "{\"lpToken\":\"LP\",\"precisionMultipliers\":[\"1\",\"1\"]}",
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
