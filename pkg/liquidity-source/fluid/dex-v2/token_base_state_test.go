package dexv2

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeployedLiquidityPermissions(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*TokenBaseState, *big.Int, *big.Int, *big.Int, *big.Int)
		want   [4]bool
	}{
		{"configured", func(*TokenBaseState, *big.Int, *big.Int, *big.Int, *big.Int) {}, [4]bool{true, true, true, true}},
		{"global pause", func(_ *TokenBaseState, _, _, _, status *big.Int) { status.SetInt64(2) }, [4]bool{}},
		{"blanket token pause", func(_ *TokenBaseState, _, _, config, _ *big.Int) { config.SetBit(config, 255, 1) }, [4]bool{}},
		{"unconfigured supply", func(_ *TokenBaseState, supply, _, _, _ *big.Int) { supply.SetInt64(0) }, [4]bool{false, false, true, true}},
		{"unconfigured borrow", func(_ *TokenBaseState, _, borrow, _, _ *big.Int) { borrow.SetInt64(0) }, [4]bool{true, true, false, false}},
		{"blanket user supply pause", func(_ *TokenBaseState, supply, _, _, _ *big.Int) { supply.SetBit(supply, 255, 1) }, [4]bool{false, false, true, true}},
		{"active decay without future credit", func(b *TokenBaseState, _, _, _, _ *big.Int) {
			b.DexSupplyAtLiquidity.DecayAmount.SetInt64(10)
			b.DexSupplyAtLiquidity.DecayEndTimestamp.SetInt64(101)
		}, [4]bool{true, true, true, true}},
		{"expired decay", func(b *TokenBaseState, _, _, _, _ *big.Int) {
			b.DexSupplyAtLiquidity.DecayAmount.SetInt64(10)
			b.DexSupplyAtLiquidity.DecayEndTimestamp.SetInt64(100)
		}, [4]bool{true, true, true, true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := TokenBaseState{DexSupplyAtLiquidity: StructsUserSupplyData{DecayAmount: new(big.Int), DecayEndTimestamp: new(big.Int)}}
			supply, borrow, config, status := big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(1)
			tc.mutate(&b, supply, borrow, config, status)
			got, err := liquidityPermissions(b, supply, borrow, config, status, 100)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
