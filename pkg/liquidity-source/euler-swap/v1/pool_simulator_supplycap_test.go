package v1

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/internal/rpcreplay"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestDepositStaysUnderTheSupplyCap tracks the Unichain WBTC/WETH pool 0x880c2a0b at block 63466002 from a fixture
// of the node's answers. Its WBTC vault (eWBTC-2) is at its supply cap: maxDeposit is 9997154959, and the pool's
// getLimits gives the same in-limit. The pool deposits the whole input, the LP fee included (less a protocol fee,
// none here), so a swap of more than that reverts with E_SupplyCapExceeded on a fork. The simulator, like the pool's
// own computeQuote, checked the in-limit against the input less the fee, and quoted inputs up to 9997254931.
//
// Set EULER_RECORD_RPC to a node at that block to record missing calls into the fixture.
func TestDepositStaysUnderTheSupplyCap(t *testing.T) {
	srv := rpcreplay.New(t, "testdata/unichain-63466002-0x880c2a0b.json", os.Getenv("EULER_RECORD_RPC"))
	defer srv.Save()
	ctx := context.Background()
	client := srv.Client()
	cfg := shared.Config{DexID: DexType}
	pools, err := NewPoolsListUpdater(&cfg, client).initPools(ctx,
		[]common.Address{common.HexToAddress("0x880c2a0b88a03817420dec5d8691a641ced3a8a8")})
	require.NoError(t, err)
	tracker, err := NewPoolTracker(&cfg, client)
	require.NoError(t, err)
	tracked, err := tracker.GetNewPoolState(ctx, pools[0], poolpkg.GetNewPoolStateParams{})
	require.NoError(t, err)
	sim, err := NewPoolSimulator(tracked)
	require.NoError(t, err)

	quote := func(amountIn int64) (*poolpkg.CalcAmountOutResult, error) {
		return sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{Token: tracked.Tokens[0].Address, Amount: big.NewInt(amountIn)},
			TokenOut:      tracked.Tokens[1].Address})
	}

	// maxDeposit itself: the pool's computeQuote gives 1949881476480380, and the swap executes on a fork.
	res, err := quote(9997154959)
	require.NoError(t, err)
	require.Equal(t, "1949881476480380", res.TokenAmountOut.Amount.String())

	// Above it the vault refuses the deposit; 9997254931 is the most the simulator used to quote.
	for _, amountIn := range []int64{9997154960 + 10, 9997254931} {
		_, err = quote(amountIn)
		require.ErrorIs(t, err, shared.ErrSwapLimitExceeded, "amountIn %d", amountIn)
	}
}
