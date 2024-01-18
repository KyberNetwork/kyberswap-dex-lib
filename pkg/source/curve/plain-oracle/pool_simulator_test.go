package plainoracle

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
	// test data from https://optimistic.etherscan.io/address/0xb90b9b1f91a01ea22a182cd84c1e22222e39b415#readContract
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"A", 100000, "B", 88639},
		{"B", 100000, "A", 112726},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"4929038393526761949570", "4622174777771844922336", "9849021650836480441313"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\", \"rates\": [%v, %v]}",
			"4000000",
			"5000000000",
			5000, 5000,
			"1000000000000000000", "1128972205632615487"),
		StaticExtra: fmt.Sprintf("{\"lpToken\": \"LP\", \"aPrecision\": \"%v\", \"precisionMultipliers\": [\"%v\", \"%v\"], \"oracle\": \"%v\"}",
			"100",
			"1", "1",
			"0xe59EBa0D492cA53C6f46015EEa00517F2707dc77"),
	})
	require.Nil(t, err)

	assert.Equal(t, 0, len(p.CanSwapTo("LP")))
	assert.Equal(t, []string{"B"}, p.CanSwapTo("A"))
	assert.Equal(t, []string{"A"}, p.CanSwapTo("B"))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
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
