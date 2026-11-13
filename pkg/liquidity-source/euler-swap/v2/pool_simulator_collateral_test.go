package v2

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/internal/rpcreplay"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestDepositIntoAVaultNotEnabledAsCollateral tracks the Base pool 0x4687e171 (cbETH/WETH) at block 55170698 from a
// fixture of the node's answers. The euler account owes cbETH and holds 0.0587 WETH; its collaterals are the WETH
// vault 0x8591 and three others, not the cbETH vault 0x358f. A cbETH-in swap above about 0.054 must borrow WETH, and
// the WETH vault's account check counts only enabled collaterals, so the chain reverts it with E_AccountLiquidity.
// The simulator counted the cbETH deposit as collateral at the WETH vault's LTV and quoted these swaps. Swaps that
// need no new borrow still quote what the pool's computeQuote gives.
//
// Set EULER_RECORD_RPC to a node at that block to record missing calls into the fixture.
func TestDepositIntoAVaultNotEnabledAsCollateral(t *testing.T) {
	srv := rpcreplay.New(t, "testdata/base-55170698-0x4687e171.json", os.Getenv("EULER_RECORD_RPC"))
	defer srv.Save()
	ctx := context.Background()
	cfg := shared.Config{DexID: DexType}
	pools, err := NewPoolsListUpdater(&cfg, srv.Client()).initPools(ctx,
		[]common.Address{common.HexToAddress("0x4687e171bea250e3048d8f426b32cb16be17a8a8")})
	require.NoError(t, err)
	tracker, err := NewPoolTracker(&cfg, srv.Client())
	require.NoError(t, err)
	tracked, err := tracker.GetNewPoolState(ctx, pools[0], pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	sim, err := NewPoolSimulator(tracked)
	require.NoError(t, err)

	const (
		cbETH = "0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22"
		weth  = "0x4200000000000000000000000000000000000006"
	)
	quote := func(in, out, amount string) (*pool.CalcAmountOutResult, error) {
		amt, _ := new(big.Int).SetString(amount, 10)
		return sim.CloneState().CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: in, Amount: amt}, TokenOut: out})
	}

	// The chain executes these; want is computeQuote at the block.
	for _, tc := range []struct{ in, out, amount, want string }{
		{cbETH, weth, "5000000000000000", "5625580272117365"},
		{cbETH, weth, "50000000000000000", "54880665374712736"},
	} {
		res, err := quote(tc.in, tc.out, tc.amount)
		require.NoError(t, err, "%s in %s", tc.in, tc.amount)
		require.Equal(t, tc.want, res.TokenAmountOut.Amount.String(), "%s in %s", tc.in, tc.amount)
	}

	// The chain reverts these with E_AccountLiquidity.
	for _, amount := range []string{"60000000000000000", "80000000000000000", "100000000000000000", "111500000000000000"} {
		_, err := quote(cbETH, weth, amount)
		require.ErrorIs(t, err, shared.ErrInsolvency, "cbETH in %s", amount)
	}
}
