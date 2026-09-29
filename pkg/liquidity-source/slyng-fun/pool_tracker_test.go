package slyngfun

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
)

func mainnetClient() *ethrpc.Client {
	return ethrpc.New(robinhoodRPC).SetMulticallContract(common.HexToAddress(robinhoodMulticall3))
}

// discoverSlyng runs the backfill pool-service drives, over the window SLYNG was created in.
// Robinhood's RPC silently truncates a getLogs answer past about a million blocks, which is why
// the scan window here, and in pool-service's config, has to stay small.
func discoverSlyng(t *testing.T) entity.Pool {
	t.Helper()

	backfiller, err := poolfactory.NewFilterLogsBackfiller(NewPoolFactoryDecoder(mainnetConfig()),
		mainnetClient(), poolfactory.BackfillConfig{
			FactoryAddresses:     []common.Address{common.HexToAddress(mainnetLaunchpad)},
			StartBlock:           slyngCreatedBlock,
			MaxBlockRangePerScan: 1,
			Exchange:             DexType,
		})
	require.NoError(t, err)

	pools, _, _, err := backfiller.Backfill(t.Context(), nil)
	require.NoError(t, err)

	p, found := lo.Find(pools, func(p entity.Pool) bool { return p.Address == slyngToken })
	require.True(t, found, "SLYNG not discovered in its own block")
	return p
}

// TestLive_SlyngOnMainnet discovers SLYNG from its creation log, tracks its curve, and checks the
// simulator's answer against the launchpad's own quoteToTokens and tokensToQuote, read in the
// same multicall as the curve so a trade landing in between shows up as a mismatch rather than
// a false pass.
func TestLive_SlyngOnMainnet(t *testing.T) {
	test.SkipCI(t)

	p := discoverSlyng(t)
	assert.Equal(t, robinhoodWETH, p.Tokens[0].Address)
	assert.EqualValues(t, slyngCreatedBlock, p.BlockNumber)

	tracker, err := NewPoolTracker(mainnetConfig(), mainnetClient())
	require.NoError(t, err)
	p, err = tracker.GetNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	var staticExtra StaticExtra
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &staticExtra))

	assert.Equal(t, slyngLaunchpad, staticExtra.Launchpad)
	assert.True(t, staticExtra.IsNativeQuote)
	assert.EqualValues(t, 100, staticExtra.TradeFeeBps)
	assert.EqualValues(t, 5000, staticExtra.SnipeBps)
	assert.EqualValues(t, 30, staticExtra.SnipeWindowSeconds)
	assert.EqualValues(t, slyngCreatedAt, staticExtra.CreatedAt)
	assert.Equal(t, "5000000000000000000", staticExtra.GraduationTarget.Dec())
	assert.Equal(t, "1500000000000000000", staticExtra.VirtualQuote.Dec())
	assert.Equal(t, extra.QuoteReserve.Dec(), p.Reserves[0])
	assert.Equal(t, extra.TokenReserve.Dec(), p.Reserves[1])
	assert.Positive(t, p.BlockNumber)
	assert.EqualValues(t, 100, p.SwapFee)
	if extra.Graduated {
		t.Skip("SLYNG has graduated; its curve no longer quotes")
	}

	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	amountIn := big.NewInt(1e16)
	net := new(big.Int).Sub(amountIn, new(big.Int).Div(amountIn, big.NewInt(100)))
	tokensIn, _ := new(big.Int).SetString("1000000000000000000000000", 10)

	var (
		curve       curveResp
		chainTokens *big.Int
		chainGross  *big.Int
		surcharge   *big.Int
	)
	token := common.HexToAddress(slyngToken)
	req := mainnetClient().NewRequest().SetContext(t.Context())
	req.AddCall(&ethrpc.Call{ABI: launchpadABI, Target: mainnetLaunchpad, Method: launchpadMethodCurves,
		Params: []any{token}}, []any{&curve})
	req.AddCall(&ethrpc.Call{ABI: launchpadABI, Target: mainnetLaunchpad, Method: "quoteToTokens",
		Params: []any{token, net}}, []any{&chainTokens})
	req.AddCall(&ethrpc.Call{ABI: launchpadABI, Target: mainnetLaunchpad, Method: "tokensToQuote",
		Params: []any{token, tokensIn}}, []any{&chainGross})
	req.AddCall(&ethrpc.Call{ABI: launchpadABI, Target: mainnetLaunchpad, Method: "snipeSurchargeBps",
		Params: []any{token}}, []any{&surcharge})
	_, err = req.Aggregate()
	require.NoError(t, err)
	require.Equal(t, extra.QuoteReserve.Dec(), curve.QuoteReserve.String(),
		"the curve moved between the two reads; run again")
	require.Zero(t, surcharge.Sign(), "SLYNG's opening window closed on launch night")

	bought, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: p.Tokens[0].Address, Amount: amountIn},
		TokenOut:      p.Tokens[1].Address,
	})
	require.NoError(t, err)
	assert.Equal(t, chainTokens.String(), bought.TokenAmountOut.Amount.String(),
		"a 0.01 ETH buy, priced by the launchpad")

	sold, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: p.Tokens[1].Address, Amount: tokensIn},
		TokenOut:      p.Tokens[0].Address,
	})
	require.NoError(t, err)
	if chainGross.Cmp(curve.QuoteReserve) > 0 {
		chainGross = curve.QuoteReserve
	}
	expected := new(big.Int).Sub(chainGross, new(big.Int).Div(chainGross, big.NewInt(100)))
	assert.Equal(t, expected.String(), sold.TokenAmountOut.Amount.String(),
		"a one million SLYNG sell, priced by the launchpad")
}

func TestLive_AnAddressTheLaunchpadNeverMinted(t *testing.T) {
	test.SkipCI(t)

	tracker, err := NewPoolTracker(mainnetConfig(), mainnetClient())
	require.NoError(t, err)
	_, err = tracker.GetNewPoolState(t.Context(), entity.Pool{Address: usdg}, pool.GetNewPoolStateParams{})
	assert.ErrorIs(t, err, ErrUnknownCurve)
}
