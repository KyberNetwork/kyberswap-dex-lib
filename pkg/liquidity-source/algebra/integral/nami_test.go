package integral

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/KyberNetwork/msgpack/v5"

	"github.com/KyberNetwork/int256"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestNamiFee(t *testing.T) {
	cfg := NamiFeeConfig{Floor: 100, Cap: 1000, Premium0For1: 100, Premium1For0: 200, MaxDiscountBps: 5000, Fee0For1: 900, Fee1For0: 100, ExpiresAt: 1000, DecayWindow: 100}
	for _, tc := range []struct {
		name string
		base uint16
		dir  bool
		now  uint64
		want uint16
	}{
		{"active high", 300, true, 999, 900}, {"active low limited discount", 300, false, 999, 250},
		{"decay high", 300, true, 1050, 650}, {"decay low", 300, false, 1050, 375},
		{"decay ended", 300, true, 1100, 400}, {"cap", 1000, true, 1100, 1000}, {"floor", 0, true, 1100, 100},
	} {
		t.Run(tc.name, func(t *testing.T) { require.Equal(t, tc.want, cfg.fee(tc.base, tc.dir, tc.now)) })
	}
	cfg.Cap = 0
	require.Equal(t, uint16(300), cfg.fee(300, true, 999))
}

func TestNamiQuoteTimepointPurity(t *testing.T) {
	oldTime := blockTimestamp
	blockTimestamp = func() uint32 { return 101 }
	t.Cleanup(func() { blockTimestamp = oldTime })
	sim := ps.CloneState().(*PoolSimulator)
	sim.globalState.PluginConfig |= BEFORE_SWAP_FLAG
	sim.volatilityOracle = &VolatilityOraclePlugin{IsInitialized: true, LastTimepointTimestamp: 100}
	sim.timepoints = NewTimepointStorage(map[uint16]Timepoint{0: {Initialized: true, BlockTimestamp: 100, Tick: sim.globalState.Tick, AverageTick: sim.globalState.Tick, VolatilityCumulative: uint256.NewInt(0)}})
	sim.dynamicFee = &DynamicFeeConfig{BaseFee: 300}
	sim.namiFee = &NamiFeeConfig{Floor: 100, Cap: 1000, Premium0For1: 100, Premium1For0: 200}
	tokens := sim.GetTokens()
	params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[0], Amount: big.NewInt(100000000000000)}, TokenOut: tokens[1]}
	first, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	second, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, first.TokenAmountOut.Amount, second.TokenAmountOut.Amount)
	require.Equal(t, uint16(0), sim.volatilityOracle.TimepointIndex)
	require.False(t, sim.timepoints.Get(1).Initialized)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: first.SwapInfo})
	require.Equal(t, uint16(1), sim.volatilityOracle.TimepointIndex)
	require.True(t, sim.timepoints.Get(1).Initialized)
	third, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	require.NotEqual(t, first.TokenAmountOut.Amount, third.TokenAmountOut.Amount)
}

// Base METAc/USDC pool 0x6dbf1772aee6f453c6dafe705bb388685d5df1ea:
// plugin swapFee(true/false) returned 1000 with a flat feeConfig and unset bounds.
func TestNamiObservedBaseFee(t *testing.T) {
	cfg := NamiFeeConfig{}
	require.Equal(t, uint16(1000), cfg.fee(1000, true, 0))
	require.Equal(t, uint16(1000), cfg.fee(1000, false, 0))
}

func TestNamiABIOutputs(t *testing.T) {
	bounds := namiFeeBounds{Floor: 100, Cap: 3000, Premium0For1: 12, Premium1For0: 45, DecayWindow: 30, MaxDiscountBps: 500}
	method := namiPluginABI.Methods["feeBounds"]
	encoded, err := method.Outputs.Pack(bounds)
	require.NoError(t, err)
	var decoded struct{ Bounds namiFeeBounds }
	require.NoError(t, namiPluginABI.UnpackIntoInterface(&decoded, "feeBounds", encoded))
	require.Equal(t, bounds, decoded.Bounds)
	original := namiFeeOverride{Fee0For1: 120, Fee1For0: 230, ExpiresAt: big.NewInt(123456789)}
	encoded, err = namiPluginABI.Methods["feeOverride"].Outputs.Pack(original)
	require.NoError(t, err)
	var override struct{ OverrideState namiFeeOverride }
	require.NoError(t, namiPluginABI.UnpackIntoInterface(&override, "feeOverride", encoded))
	require.Equal(t, original, override.OverrideState)
}

// Historical Base swaps of the METAc/USDC pool, using state from the preceding block.
// No initialized tick lies between the starting and ending ticks for either swap.
func TestNamiBaseSwapReceipts(t *testing.T) {
	oldTime := blockTimestamp
	t.Cleanup(func() { blockTimestamp = oldTime })
	for _, tc := range []struct {
		tx            string
		timestamp     uint32
		price         string
		tick          int32
		direction     int
		input, output int64
	}{
		{"0x4207ca5dbb2fc85e6eff3b41ed2eb3eb7727a503a4213712b729de4e974c7bee", 1793598911, "31900346563188641650020041102", -18196, 0, 4505817, 728168},
		{"0xffcb45b01a288e51ac6a1ca9ddf654b66770867fb733e41f28090db1f31217ae", 1793602535, "31831480227066460847131138237", -18239, 1, 763165, 4712445},
	} {
		t.Run(tc.tx, func(t *testing.T) {
			blockTimestamp = func() uint32 { return tc.timestamp }
			sim := ps.CloneState().(*PoolSimulator)
			sim.globalState = GlobalState{Price: uint256.MustFromDecimal(tc.price), Tick: tc.tick, LastFee: 500, PluginConfig: 195, CommunityFee: 1000, Unlocked: true}
			sim.liquidity = uint256.NewInt(837731207)
			ticks, err := v3Entities.NewTickListDataProvider([]v3Entities.Tick{
				{Index: -18540, LiquidityGross: uint256.NewInt(1), LiquidityNet: int256.NewInt(1)},
				{Index: -17800, LiquidityGross: uint256.NewInt(1), LiquidityNet: int256.NewInt(-1)},
			}, 1)
			require.NoError(t, err)
			sim.ticks = ticks
			sim.tickMin = -18540
			sim.tickMax = -17800
			sim.volatilityOracle = &VolatilityOraclePlugin{IsInitialized: true, LastTimepointTimestamp: tc.timestamp}
			sim.timepoints = NewTimepointStorage(map[uint16]Timepoint{0: {Initialized: true, BlockTimestamp: tc.timestamp, Tick: tc.tick, AverageTick: tc.tick, VolatilityCumulative: uint256.NewInt(0)}})
			sim.dynamicFee = &DynamicFeeConfig{BaseFee: 1000}
			sim.namiFee = &NamiFeeConfig{}
			tokens := sim.GetTokens()
			result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[tc.direction], Amount: big.NewInt(tc.input)}, TokenOut: tokens[1-tc.direction]})
			require.NoError(t, err)
			require.Equal(t, big.NewInt(tc.output), result.TokenAmountOut.Amount)
		})
	}
}

func TestNamiStateMsgpackRoundTrip(t *testing.T) {
	oldTime := blockTimestamp
	blockTimestamp = func() uint32 { return 101 }
	t.Cleanup(func() { blockTimestamp = oldTime })
	sim := ps.CloneState().(*PoolSimulator)
	sim.globalState.PluginConfig |= BEFORE_SWAP_FLAG
	sim.volatilityOracle = &VolatilityOraclePlugin{IsInitialized: true, LastTimepointTimestamp: 100}
	sim.timepoints = NewTimepointStorage(map[uint16]Timepoint{0: {Initialized: true, BlockTimestamp: 100, Tick: sim.globalState.Tick, AverageTick: sim.globalState.Tick, VolatilityCumulative: uint256.NewInt(0)}})
	sim.dynamicFee = &DynamicFeeConfig{BaseFee: 300}
	sim.namiFee = &NamiFeeConfig{Floor: 100, Cap: 1000, Premium0For1: 100, Premium1For0: 200}
	tokens := sim.GetTokens()
	params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[0], Amount: big.NewInt(100000000000000)}, TokenOut: tokens[1]}
	first, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: first.SwapInfo})
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.IncludeUnexported(true)
	enc.SetForceAsArray(true)
	require.NoError(t, enc.Encode(sim))
	dec := msgpack.NewDecoder(&buf)
	dec.IncludeUnexported(true)
	dec.SetForceAsArray(true)
	var restored PoolSimulator
	require.NoError(t, dec.Decode(&restored))
	want, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	got, err := restored.CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, want.TokenAmountOut.Amount, got.TokenAmountOut.Amount)
}

func TestNamiUnsupportedStateCannotQuote(t *testing.T) {
	sim := ps.CloneState().(*PoolSimulator)
	sim.namiFee = &NamiFeeConfig{Unsupported: "Nami external fee source is not supported"}
	tokens := sim.GetTokens()
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[0], Amount: big.NewInt(1000000)}, TokenOut: tokens[1]})
	require.ErrorContains(t, err, "Nami external fee source")
}
