package lido

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	// test data from https://etherscan.io/address/0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0#readContract
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"stETH", 100, "wstETH", 88},
		{"wstETH", 100, "stETH", 112},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"2264571555224494676557305", "2005870067403083354670050"},
		Tokens:   []*entity.PoolToken{{Address: "stETH"}, {Address: "wstETH"}},
		Extra: fmt.Sprintf("{\"stEthPerToken\": %v, \"tokensPerStEth\": %v}",
			"1128972205632615487",
			"885761398740240572"),
		StaticExtra: "{\"lpToken\": \"wstETH\"}",
	})
	require.Nil(t, err)

	assert.Equal(t, []string{"wstETH"}, p.CanSwapTo("stETH"))
	assert.Equal(t, []string{"stETH"}, p.CanSwapTo("wstETH"))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := p.CalcAmountOut(pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}, tc.out)
			require.Nil(t, err)
			assert.Equal(t, big.NewInt(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
		})
	}
}
