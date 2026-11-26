package v2

import (
	"context"
	"encoding/hex"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/internal/rpcreplay"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestPinnedReadStaysAtThePin: a read pinned to a block is dated by it, and its later rounds are
// read there, though the first round's multicall reports the block after, as Etherlink's does.
func TestPinnedReadStaysAtThePin(t *testing.T) {
	tr, err := NewPoolTracker(&shared.Config{}, ethrpc.New("http://127.0.0.1:1"))
	require.NoError(t, err)

	later := map[int]*big.Int{}
	exec := func(req *ethrpc.Request, round int) (*big.Int, error) {
		if round == 0 {
			return new(big.Int).Add(req.BlockNumber, big.NewInt(1)), nil
		}
		later[round] = req.BlockNumber
		return nil, nil
	}
	vaults := []shared.VaultInfo{
		{VaultAddress: "0x1", AssetAddress: "0xa"},
		{VaultAddress: "0x2", AssetAddress: "0xb"},
	}
	_, bn, err := tr.getPoolData(t.Context(), "0xp", "0xe", "0xc", vaults, nil, big.NewInt(100), exec)
	require.NoError(t, err)
	assert.EqualValues(t, 100, bn)
	require.Contains(t, later, 3)
	assert.EqualValues(t, 100, later[3].Uint64())
}

// TestTrackVaultWithoutOracle tracks the Ethereum pool 0x0225eac3 at block 26340400 from a fixture of the node's
// answers. A vault without an oracle gets no price calls; the tracker used to grow each collateral's price row only up
// to the last vault with one, and indexed past its end (index out of range) when a later vault had none.
//
// Set EULER_RECORD_RPC to a node at that block to record missing calls into the fixture.
func TestTrackVaultWithoutOracle(t *testing.T) {
	srv := rpcreplay.New(t, "testdata/ethereum-26340400-0x0225eac3.json", os.Getenv("EULER_RECORD_RPC"))
	defer srv.Save()
	ctx := context.Background()
	client := srv.Client()
	cfg := shared.Config{DexID: DexType}
	pools, err := NewPoolsListUpdater(&cfg, client).initPools(ctx,
		[]common.Address{common.HexToAddress("0x0225eac3d800a8de0954dd046d13dd21546228a8")})
	require.NoError(t, err)
	tracker, err := NewPoolTracker(&cfg, client)
	require.NoError(t, err)
	tracked, err := tracker.GetNewPoolState(ctx, pools[0], pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	sim, err := NewPoolSimulator(tracked)
	require.NoError(t, err)

	// The pool's computeQuote at that block, for 0.001 of each token.
	const (
		weth  = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
		token = "0xe1b4d34e8754600962cd944b535180bd758e6c2e"
	)
	for _, tc := range []struct{ in, out, amount, want string }{
		{weth, token, "1000000000000000", "948745005077017"},
		{token, weth, "1000000000000000", "1053813200227429"},
	} {
		amount, _ := new(big.Int).SetString(tc.amount, 10)
		res, err := sim.CloneState().CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: amount}, TokenOut: tc.out})
		if err != nil {
			t.Errorf("%s -> %s %s: %v", tc.in, tc.out, tc.amount, err)
			continue
		}
		require.Equal(t, tc.want, res.TokenAmountOut.Amount.String(), "%s -> %s %s", tc.in, tc.out, tc.amount)
	}
}

// fakeExec answers a getPoolData round without a node: the account has controller 0xc1 (not one of the pool's
// vaults) and one collateral 0xc2 (not a vault either), and every vault has an oracle. It records each call's
// method, and each getQuotes call's inAmount.
type fakeExec struct {
	methods []string
	amounts []*big.Int
}

func (f *fakeExec) exec(req *ethrpc.Request, round int) (*big.Int, error) {
	for _, c := range req.Calls {
		f.methods = append(f.methods, c.Method)
		if c.Method == shared.RouterMethodGetQuotes {
			f.amounts = append(f.amounts, c.Params[0].(*big.Int))
		}
		switch out := c.Output[0].(type) {
		case *common.Address:
			switch c.Method {
			case shared.VaultMethodOracle:
				*out = common.HexToAddress("0xaa")
			case shared.VaultMethodUnitOfAccount:
				*out = common.HexToAddress("0xbb")
			case shared.VaultMethodAsset:
				*out = common.HexToAddress("0xdd")
			}
		case *[]common.Address:
			switch c.Method {
			case shared.EvcMethodGetControllers:
				*out = []common.Address{common.HexToAddress("0xc1")}
			case shared.EvcMethodGetCollaterals:
				*out = []common.Address{common.HexToAddress("0xc2")}
			}
		}
	}
	return big.NewInt(100), nil
}

// TestGetPoolDataQuotesTheFixedAmount: every oracle quote getPoolData asks for, a pool's vault, the controller
// and a collateral outside the pool alike, is for shared.QuoteAmount, and no round reads decimals().
func TestGetPoolDataQuotesTheFixedAmount(t *testing.T) {
	tr, err := NewPoolTracker(&shared.Config{}, ethrpc.New("http://127.0.0.1:1"))
	require.NoError(t, err)
	vaults := []shared.VaultInfo{
		{VaultAddress: "0x1", AssetAddress: "0xa"},
		{VaultAddress: "0x2", AssetAddress: "0xb"},
	}

	var f fakeExec
	_, _, err = tr.getPoolData(t.Context(), "0xp", "0xe", "0xc", vaults, nil, nil, f.exec)
	require.NoError(t, err)

	// 3 vaults (2 + the controller) and 1 collateral, each quoted by 3 oracles.
	require.Len(t, f.amounts, 3*(3+1))
	for i, a := range f.amounts {
		assert.Equal(t, shared.QuoteAmount.String(), a.String(), "quote %d", i)
	}
	assert.NotContains(t, f.methods, "decimals", "a round reads decimals()")
}

// TestPricesQuoteTheFixedAmount: the pool's tokens carry no decimals, so quoting 1 smallest unit of each asset lost
// the oracle's fraction (2315 for WETH at $2315.96). The tracker quotes shared.QuoteAmount, whatever the asset's
// decimals, and prices come out per whole unit as before: the oracle's getQuotes(1e18, WETH, USD) at the Ethereum
// pool 0x0225eac3, block 26340400, is 2315957979270000000000, and QuoteAmount is the same 1e18, so the fixture's quote is
// recorded. Not every fixture entry is: see the commit message. Neither the
// single read nor the batch read asks a vault for its decimals.
func TestPricesQuoteTheFixedAmount(t *testing.T) {
	srv := rpcreplay.New(t, "testdata/ethereum-26340400-0x0225eac3.json", "")
	ctx := context.Background()
	cfg := shared.Config{DexID: DexType}
	pools, err := NewPoolsListUpdater(&cfg, srv.Client()).initPools(ctx,
		[]common.Address{common.HexToAddress("0x0225eac3d800a8de0954dd046d13dd21546228a8")})
	require.NoError(t, err)
	tracker, err := NewPoolTracker(&cfg, srv.Client())
	require.NoError(t, err)
	tracked, err := tracker.GetNewPoolState(ctx, pools[0], pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	// A batch read without the lens reads each pool alone, and quotes shared.QuoteAmount as well.
	batch, errs := tracker.GetNewPoolStates(ctx, pools[:1])
	require.NoError(t, errs[0])

	for name, p := range map[string]string{"single": tracked.Extra, "batch": batch[0].Extra} {
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(p), &extra), name)
		weth := extra.SupplyVault[0]
		require.Equal(t, "2315957979270000000000", weth.DebtPrice.Dec(), name)
		require.Equal(t, "2315957979270000000000", weth.VaultValuePrices[0].Dec(), name)
	}

	assert.Zero(t, srv.Selectors()[hex.EncodeToString(vaultABI.Methods["decimals"].ID)], "decimals() was asked")
	amounts := srv.QuotedAmounts()
	require.NotEmpty(t, amounts)
	for _, a := range amounts {
		assert.Equal(t, shared.QuoteAmount.String(), a.String())
	}
}
