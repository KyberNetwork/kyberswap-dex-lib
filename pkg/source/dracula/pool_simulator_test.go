package dracula

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestCalcAmountOut(t *testing.T) {
	//token0: 0x3355df6D4c9C3035724Fd0e3914dE96A5a83aaf4 (USDC=>A)
	//token1: 0x5AEa5775959fBC2557Cc8789bC1bf90A239D9a91 (WETH=>B)
	//test data from https://explorer.zksync.io/address/0xB51E60f61c48d8329843F86d861AbF50E4DC918d
	testcases := []struct {
		in                string
		inAmount          string
		out               string
		expectedOutAmount string
	}{
		{"A", "100", "B", "58175411896"},
		{"B", "100000000000", "A", "171"},
	}

	p, err := NewPoolSimulator(entity.Pool{
		Exchange:    "",
		Type:        "",
		SwapFee:     0.0025,
		Reserves:    entity.PoolReserves{"20694815319", "12039294111262365027"},
		Tokens:      []*entity.PoolToken{{Address: "A", Decimals: 6}, {Address: "B", Decimals: 18}},
		StaticExtra: "{\"stable\": false}",
	})
	require.Nil(t, err)

	assert.Equal(t, []string{"A"}, p.CanSwapTo("B"))
	assert.Equal(t, []string{"B"}, p.CanSwapTo("A"))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			amountIn := pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)}
			out, err := p.CalcAmountOut(amountIn, tc.out)
			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
		})
	}
}
