package ramsesv2

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

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	// Tx simulate: https://www.tdly.co/shared/simulation/30202958-4fb6-4144-bda4-4099eea6be11
	token0 := "0x912ce59144191c1204e64559fe8253a0e49e6548"
	token1 := "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8"

	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{token0, 1000000000000000000, token1, 486457},
	}
	p, err := NewPoolSimulator(
		entity.Pool{
			Exchange: "ramses-v2",
			Type:     "ramses-v2",
			SwapFee:  500,
			Reserves: entity.PoolReserves{"69893656923366160706", "2169623"},
			Tokens:   []*entity.PoolToken{{Address: token0, Decimals: 18}, {Address: token1, Decimals: 6}},
			Extra:    "{\"liquidity\":481329773989005,\"sqrtPriceX96\":55312754561266099398800,\"feeTier\":500,\"tickSpacing\":10,\"tick\":-283511,\"ticks\":[{\"index\":-887270,\"liquidityGross\":106514621957,\"liquidityNet\":106514621957},{\"index\":-283610,\"liquidityGross\":312504599701008,\"liquidityNet\":312504599701008},{\"index\":-283580,\"liquidityGross\":168718659666040,\"liquidityNet\":168718659666040},{\"index\":-283380,\"liquidityGross\":17166285019404,\"liquidityNet\":17166285019404},{\"index\":-282780,\"liquidityGross\":481223259367048,\"liquidityNet\":-481223259367048},{\"index\":-278550,\"liquidityGross\":7351763429974,\"liquidityNet\":7351763429974},{\"index\":-276330,\"liquidityGross\":7351763429974,\"liquidityNet\":-7351763429974},{\"index\":-276170,\"liquidityGross\":294632869974088,\"liquidityNet\":294632869974088},{\"index\":-275820,\"liquidityGross\":22619085245,\"liquidityNet\":22619085245},{\"index\":-274030,\"liquidityGross\":294632869974088,\"liquidityNet\":-294632869974088},{\"index\":-269510,\"liquidityGross\":17166285019404,\"liquidityNet\":-17166285019404},{\"index\":887270,\"liquidityGross\":129133707202,\"liquidityNet\":-129133707202}],\"unlocked\":true}",
		}, 1)
	require.Nil(t, err)

	assert.Equal(t, []string{token1}, p.CanSwapTo(token0))
	assert.Equal(t, []string{token0}, p.CanSwapTo(token1))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			in := pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: in,
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
