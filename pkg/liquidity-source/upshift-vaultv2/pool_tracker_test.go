package upshiftvaultv2

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
)

// Live check: tracker + simulator vs vault.previewDeposit at the tracker's block.
// UPSHIFT_VAULTV2_RPC and UPSHIFT_VAULTV2_VAULT select the chain and vault (default: monad earnAUSD).
func TestPoolTracker_MatchesPreviewDeposit(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	rpc, vault := os.Getenv("UPSHIFT_VAULTV2_RPC"), os.Getenv("UPSHIFT_VAULTV2_VAULT")
	if rpc == "" || vault == "" {
		t.Skip("UPSHIFT_VAULTV2_RPC / UPSHIFT_VAULTV2_VAULT not set")
	}
	client := ethrpc.New(rpc).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	cfg := &Config{DexID: DexType, Vaults: []string{vault}}
	ctx := context.Background()

	pools, _, err := NewPoolsListUpdater(cfg, client).GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	tracker, err := NewPoolTracker(cfg, client)
	require.NoError(t, err)
	p, err := tracker.GetNewPoolState(ctx, pools[0], pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	t.Logf("block %d extra %s reserves %v", p.BlockNumber, p.Extra, p.Reserves)

	asset, lp := p.Tokens[0].Address, p.Tokens[1].Address
	for _, a := range []string{"1000", "999999", "1000000", "123456789", "1000000000000", "5000000000000000000000", "999999999999999999"} {
		amount, _ := new(big.Int).SetString(a, 10)
		var out struct {
			Shares                  *big.Int `abi:"shares"`
			AmountInReferenceTokens *big.Int `abi:"amountInReferenceTokens"`
		}
		_, err := client.NewRequest().SetContext(ctx).SetBlockNumber(new(big.Int).SetUint64(p.BlockNumber)).
			AddCall(&ethrpc.Call{ABI: vaultABI, Target: vault, Method: "previewDeposit",
				Params: []any{common.HexToAddress(asset), amount}}, []any{&out}).Call()
		require.NoError(t, err)
		res, serr := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: asset, Amount: amount}, TokenOut: lp})
		if sim.DepositsPaused {
			assert.ErrorIs(t, serr, ErrDepositsPaused)
			continue
		}
		if serr != nil {
			t.Logf("amount %s: sim err %v", a, serr)
			continue
		}
		assert.Equal(t, out.Shares.String(), res.TokenAmountOut.Amount.String(), "amount %s", a)
	}
}
