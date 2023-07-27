package algebrav1

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	// test data from https://polygonscan.com/address/0xd372b5067fe9cbac932af47406fdb9c64666295b#readContract
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"A", 10, "B", 11815484110223},
		{"B", 100000000000000000, "A", 75997},
	}
	g := GlobalState{
		Price:              bignumber.NewBig10("90778731131334971326752767343040037"),
		Tick:               big.NewInt(279046),
		Fee:                0,
		TimepointIndex:     62,
		CommunityFeeToken0: 0,
		CommunityFeeToken1: 0,
		Unlocked:           true,
	}
	gs, _ := json.Marshal(g)
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"1156075", "35450062374042037833"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    fmt.Sprintf("{\"liquidity\":119137538372759,\"volumePerLiquidityInBlock\":100,\"totalFeeGrowth0Token\":303633474589870761058966414024, \"totalFeeGrowth1Token\":423897051166683508742054049450199029991046,\"globalState\": %s,\"ticks\":[{\"index\":-887220,\"liquidityGross\":2822091172725,\"liquidityNet\":2822091172725},{\"index\":273540,\"liquidityGross\":116315447200034,\"liquidityNet\":116315447200034},{\"index\":279120,\"liquidityGross\":116315447200034,\"liquidityNet\":-116315447200034},{\"index\":285480,\"liquidityGross\":2822091172725,\"liquidityNet\":-2822091172725},{\"index\":887220,\"liquidityGross\":0,\"liquidityNet\":0}]}", gs),
	}, valueobject.ChainIDPolygon)
	require.Nil(t, err)

	assert.Equal(t, []string{"A"}, p.CanSwapTo("B"))
	assert.Equal(t, []string{"B"}, p.CanSwapTo("A"))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := p.CalcAmountOut(pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}, tc.out)
			require.Nil(t, err)
			assert.Equal(t, big.NewInt(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
		})
	}
}
