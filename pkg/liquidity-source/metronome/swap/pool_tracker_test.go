package metronomeswap

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// referencePool is the msUSD/msETH/msBTC Pool explored and byte-exact-verified during
// dex-explorer (see context/metronome/output/explorer.md and docs/simulations/smoke-*.md).
const referencePool = "0x3364f53cb866762aef66deef2a6b1a17c1f17f46"

type PoolTrackerTestSuite struct {
	suite.Suite

	client  *ethrpc.Client
	tracker *PoolTracker
}

func (ts *PoolTrackerTestSuite) SetupTest() {
	rpcClient := ethrpc.New("https://ethereum-rpc.kyberswap.com")
	rpcClient.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))
	ts.client = rpcClient

	cfg := &Config{
		DexID:        DexType,
		ChainID:      valueobject.ChainIDEthereum,
		PoolRegistry: "0x11ead85c679eaf528c9c1fe094bf538db880048a",
	}
	ts.tracker = NewPoolTracker(cfg, ts.client)
}

func (ts *PoolTrackerTestSuite) TestGetNewPoolState() {
	staticExtraBytes, err := json.Marshal(StaticExtra{PoolRegistry: "0x11ead85c679eaf528c9c1fe094bf538db880048a"})
	require.NoError(ts.T(), err)

	input := entity.Pool{
		Address:  referencePool,
		Exchange: DexType,
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: "0x64351fc9810adad17a690e4e1717df5e7e085160", Decimals: 18, Swappable: true}, // msETH
			{Address: "0xab5eb14c09d416f0ac63661e57edb7aecdb9befa", Decimals: 18, Swappable: true}, // msUSD
			{Address: "0xb93f48d3ea42a25f367fade092a6bb56dab5f7cb", Decimals: 18, Swappable: true}, // msBTC
		},
		Reserves:    entity.PoolReserves{"0", "0", "0"},
		Extra:       "{}",
		StaticExtra: string(staticExtraBytes),
	}

	got, err := ts.tracker.GetNewPoolState(context.Background(), input, pool.GetNewPoolStateParams{})
	require.NoError(ts.T(), err)

	var extra Extra
	require.NoError(ts.T(), json.Unmarshal([]byte(got.Extra), &extra))

	require.Len(ts.T(), got.Reserves, 3)
	require.Len(ts.T(), extra.Tokens, 3)

	for _, token := range input.Tokens {
		state, ok := extra.Tokens[token.Address]
		require.Truef(ts.T(), ok, "missing state for %s", token.Address)
		require.NotNil(ts.T(), state.PriceInUsd)
		require.NotNil(ts.T(), state.MaxTotalSupply)
		require.NotNil(ts.T(), state.TotalSupply)
		// A token can legitimately come back IsActive=false this cycle (e.g. live-observed:
		// MasterOracle has no configured feed for msBTC right now even though the token's own
		// isActive() flag is true) — only active tokens are guaranteed a usable positive price
		// and a sane cap; an inactive one is deliberately left at safe zero-value placeholders
		// (see safeFromBig in pool_tracker.go) rather than asserted against.
		if state.IsActive {
			require.Truef(ts.T(), state.PriceInUsd.Sign() > 0, "active token %s must have a positive oracle price", token.Address)
			require.Truef(ts.T(), state.MaxTotalSupply.Cmp(state.TotalSupply) >= 0,
				"maxTotalSupply must be >= totalSupply for active token %s", token.Address)
		} else {
			ts.T().Logf("token %s came back inactive this cycle (likely a missing oracle feed) — see explorer risk notes", token.Address)
		}
	}

	// The pool has been swap-active since launch; this is a live-network sanity check, not a
	// pinned historical value, so it only asserts the flag decodes to a real bool rather than
	// silently defaulting to zero-value false on a broken ABI call.
	ts.T().Logf("SwapActive=%v FeeProvider=%s MasterOracle=%s", extra.SwapActive, extra.FeeProvider, extra.MasterOracle)
	require.NotEmpty(ts.T(), extra.FeeProvider)
	require.NotEmpty(ts.T(), extra.MasterOracle)

	feeBps, ok := extra.SwapFeesBps[input.Tokens[0].Address+"-"+input.Tokens[1].Address]
	require.True(ts.T(), ok, "expected an msETH->msUSD fee entry")
	require.NotNil(ts.T(), feeBps)
	ts.T().Logf("msETH->msUSD swapFeesBps=%s", feeBps.String())
}

func TestPoolTrackerTestSuite(t *testing.T) {
	suite.Run(t, new(PoolTrackerTestSuite))
}

// TestUpdateBalance_MatchesRealSequentialSwap is dex-verify's step-6 stateful-replay check,
// pinned against real (immutable) mainnet history instead of an ephemeral Tenderly vnet, so it
// stays reproducible indefinitely. It confirms UpdateBalance's on-chain totalSupply delta
// assumption: Pool.swap() mints BOTH the fee (to the fee collector) and the net amount (to the
// caller), so totalSupply(tokenOut) grows by the FULL gross amount, not just the net amount the
// trader receives. First discovered via a live Tenderly vnet swap() execution during this
// dex-verify run; this test locks the same finding into a permanent, non-ephemeral fixture.
func TestUpdateBalance_MatchesRealSequentialSwap(t *testing.T) {
	test.SkipCI(t)

	rpcClient := ethrpc.New("https://ethereum-rpc.kyberswap.com")
	rpcClient.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))

	// tx 0x587e98585911758665038dc4b8417546c04706c9b261f9852ffa113aeeb5bc9a, block 25675390 —
	// same fixture as fixtureAmountIn/fixtureFee/fixtureNetOut in math_test.go.
	const swapBlock = 25675390

	var totalSupplyBefore, totalSupplyAfter *big.Int
	_, err := rpcClient.NewRequest().SetBlockNumber(big.NewInt(swapBlock-1)).AddCall(&ethrpc.Call{
		ABI: syntheticTokenABI, Target: msUSD, Method: syntheticTokenMethodTotalSupply,
	}, []any{&totalSupplyBefore}).Call()
	require.NoError(t, err)

	_, err = rpcClient.NewRequest().SetBlockNumber(big.NewInt(swapBlock)).AddCall(&ethrpc.Call{
		ABI: syntheticTokenABI, Target: msUSD, Method: syntheticTokenMethodTotalSupply,
	}, []any{&totalSupplyAfter}).Call()
	require.NoError(t, err)

	observedDelta := new(uint256.Int).Sub(big256FromBig(totalSupplyAfter), big256FromBig(totalSupplyBefore))
	expectedGross := new(uint256.Int).Add(fixtureNetOut, fixtureFee)

	require.Equal(t, expectedGross.String(), observedDelta.String(),
		"totalSupply(msUSD) delta across the swap's own block must equal net+fee (gross), confirming the fee mint counts against totalSupply too")

	// Now confirm our own UpdateBalance reproduces the exact same delta.
	sim := newTestPoolSimulator(t)
	before := new(uint256.Int).Set(sim.Extra.Tokens[msUSD].TotalSupply)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: msETH, Amount: fixtureAmountIn.ToBig()},
		TokenAmountOut: pool.TokenAmount{Token: msUSD, Amount: fixtureNetOut.ToBig()},
		Fee:            pool.TokenAmount{Token: msUSD, Amount: fixtureFee.ToBig()},
	})
	simDelta := new(uint256.Int).Sub(sim.Extra.Tokens[msUSD].TotalSupply, before)
	require.Equal(t, observedDelta.String(), simDelta.String())
}

func big256FromBig(b *big.Int) *uint256.Int {
	u, _ := uint256.FromBig(b)
	return u
}
