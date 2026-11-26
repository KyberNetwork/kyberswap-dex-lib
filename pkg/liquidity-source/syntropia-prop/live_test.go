package syntropiaprop

import (
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const multicall3 = "0xcA11bde05977b3631167028862bE2a173976CA11"

// TestLive lists the venue's pairs, tracks each, and compares the simulator with Pool.quote
// at the tracked block. Run it with
//
//	SYNTROPIA_PROP_RPC=<base rpc> go test -run TestLive -v ./pkg/liquidity-source/syntropia-prop/
//
// SYNTROPIA_PROP_POOL and SYNTROPIA_PROP_LENS override the Base addresses.
func TestLive(t *testing.T) {
	url := os.Getenv("SYNTROPIA_PROP_RPC")
	if url == "" {
		t.Skip("set SYNTROPIA_PROP_RPC to compare with the pool on chain")
	}
	cfg := &Config{DexID: DexType, Pool: testPool, Lens: testLens, Buffer: 10_000}
	if v := os.Getenv("SYNTROPIA_PROP_POOL"); v != "" {
		cfg.Pool = v
	}
	if v := os.Getenv("SYNTROPIA_PROP_LENS"); v != "" {
		cfg.Lens = v
	}
	client := ethrpc.New(url).SetMulticallContract(common.HexToAddress(multicall3))
	erc20, err := abi.JSON(strings.NewReader(`[{"type":"function","name":"decimals","inputs":[],"outputs":[{"type":"uint8"}],"stateMutability":"view"}]`))
	require.NoError(t, err)

	lister, err := NewPoolsListUpdater(cfg, client)
	require.NoError(t, err)
	pools, _, err := lister.GetNewPools(t.Context(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools)
	tracker, err := NewPoolTracker(cfg, client)
	require.NoError(t, err)

	for _, p := range pools {
		for _, token := range p.Tokens {
			_, err := client.NewRequest().SetContext(t.Context()).AddCall(&ethrpc.Call{
				ABI: erc20, Target: token.Address, Method: "decimals",
			}, []any{&token.Decimals}).Call()
			require.NoError(t, err)
		}
		p, err = tracker.GetNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		simulator, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
		require.NoError(t, err)
		var extra orderbook.Extra
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
		t.Logf("%s block %d reserves %v", p.Address, p.BlockNumber, p.Reserves)

		for dir := range 2 {
			tokenIn, tokenOut := p.Tokens[dir], p.Tokens[1-dir]
			depth := 0.0
			for _, level := range extra.LevelsFrom[dir] {
				depth += level.Size()
			}
			if depth == 0 {
				_, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tokenIn.Address, Amount: big.NewInt(1_000_000)},
					TokenOut:      tokenOut.Address,
				})
				require.Error(t, err)
				t.Logf("  %s -> %s: no levels, not quoted", tokenIn.Address, tokenOut.Address)
				continue
			}
			raw, _ := new(big.Float).Mul(big.NewFloat(depth), new(big.Float).SetInt(
				new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(tokenIn.Decimals)), nil))).Int(nil)
			for _, perMille := range []int64{1, 10, 70, 300, 600, 950} {
				amountIn := new(big.Int).Div(new(big.Int).Mul(raw, big.NewInt(perMille)), big.NewInt(1000))
				var onChain *big.Int
				_, err := client.NewRequest().SetContext(t.Context()).SetBlockNumber(new(big.Int).SetUint64(p.BlockNumber)).
					AddCall(&ethrpc.Call{
						ABI: poolABI, Target: cfg.Pool, Method: "quote",
						Params: []any{common.HexToAddress(tokenIn.Address), common.HexToAddress(tokenOut.Address), amountIn},
					}, []any{&onChain}).Call()
				require.NoError(t, err)
				result, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tokenIn.Address, Amount: amountIn},
					TokenOut:      tokenOut.Address,
				})
				require.NoError(t, err)
				got := result.TokenAmountOut.Amount
				under := new(big.Int).Sub(onChain, got)
				underBps, _ := new(big.Float).Quo(new(big.Float).SetInt(new(big.Int).Mul(under, big.NewInt(10_000))),
					new(big.Float).SetInt(onChain)).Float64()
				t.Logf("  %s -> %s in %s: pool %s, simulator %s (%.3f bps under)",
					tokenIn.Address, tokenOut.Address, amountIn, onChain, got, underBps)
				require.GreaterOrEqual(t, under.Sign(), 0, "simulator quoted above the pool")
				require.LessOrEqual(t, underBps, 5.0)
			}
		}
	}
}
