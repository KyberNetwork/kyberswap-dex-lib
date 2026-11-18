package nightlyprop

import (
	"bytes"
	"math/big"
	"testing"
	"time"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/nightly-prop/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const simRouter = "0xede7b21238ab8d2bc7c5e62d7e6d141caba5cba2"

func units(tb testing.TB, amount uint64, decimals uint8) uint256.Int {
	tb.Helper()
	z := new(uint256.Int).Mul(uint256.NewInt(amount), new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(uint64(decimals))))
	return *z
}

func simLevel(lower, upper math.SignedSpread, base, quote uint256.Int) math.Level {
	return math.Level{
		Config:    math.PriceLevelConfig{LowerOffset: lower, UpperOffset: upper},
		Liquidity: math.Liquidity{BaseAmount: base, QuoteAmount: quote},
	}
}

// simExtra is the walk fixture of the math package on the WETH/USDC pool: 20,000 USDC in level
// -1 (bid 1,998 to 2,000), 10 WETH in level 0 (ask 2,000 to 2,002), and 20 WETH in level 1, the
// position at the lower edge of level 0, and WETH at $2,000 with zero oracle spreads. The price
// has the time priceTimestampMs and a 30-second maximum age. Every role and the partner are set.
func simExtra(tb testing.TB, priceTimestampMs uint64) Extra {
	return Extra{
		Routed:                 true,
		RouterSwapper:          true,
		PoolPriceReader:        true,
		PartnerRole:            true,
		PartnerFeeRecipientSet: true,
		PriceAvailable:         true,
		BlockTimestamp:         priceTimestampMs / 1_000,
		Price:                  math.OraclePrice{MidPrice: 200_000_000_000, TimestampMs: priceTimestampMs},
		Pool: math.Pool{
			BaseDecimals:  18,
			QuoteDecimals: 6,
			MaxPriceAgeMs: 30_000,
			LowestLevelID: -1,
			Levels: []math.Level{
				simLevel(-1_000, 0, uint256.Int{}, units(tb, 20_000, 6)),
				simLevel(0, 1_000, units(tb, 10, 18), uint256.Int{}),
				simLevel(1_000, 2_000, units(tb, 20, 18), uint256.Int{}),
			},
			Position: math.Position{LevelID: 0, Fraction: math.Lower},
		},
	}
}

func nowMs() uint64 {
	return uint64(time.Now().UnixMilli())
}

func newSim(tb testing.TB, extra Extra, staleCheck bool) *PoolSimulator {
	tb.Helper()
	extraBytes, err := json.Marshal(&extra)
	require.NoError(tb, err)
	sim, err := NewPoolSimulator(pool.FactoryParams{
		EntityPool: entity.Pool{
			Address:     arbitrumWethUsdcPool,
			Exchange:    DexType,
			Type:        DexType,
			Tokens:      []*entity.PoolToken{{Address: arbitrumWeth, Swappable: true}, {Address: arbitrumUsdc, Swappable: true}},
			Reserves:    extra.reserves(),
			Extra:       string(extraBytes),
			StaticExtra: `{"routerAddress":"` + simRouter + `"}`,
			BlockNumber: 525_043_339,
		},
		Opts: pool.FactoryOpts{StaleCheck: staleCheck},
	})
	require.NoError(tb, err)
	return sim
}

func quote(sim *PoolSimulator, tokenIn string, amountIn *uint256.Int, tokenOut string) (*pool.CalcAmountOutResult, error) {
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn.ToBig()},
		TokenOut:      tokenOut,
	})
}

func usdcUnits(tb testing.TB, amount uint64) *uint256.Int {
	z := units(tb, amount, 6)
	return &z
}

func wethUnits(tb testing.TB, amount uint64) *uint256.Int {
	z := units(tb, amount, 18)
	return &z
}

// TestCalcAmountOut_Buy: a USDC input buys WETH with the math walk; 7,000 USDC from the lower edge
// of level 0 is the math package's vector.
func TestCalcAmountOut_Buy(t *testing.T) {
	sim := newSim(t, simExtra(t, nowMs()), true)
	result, err := quote(sim, arbitrumUsdc, usdcUnits(t, 7_000), arbitrumWeth)
	require.NoError(t, err)
	require.Equal(t, arbitrumWeth, result.TokenAmountOut.Token)
	require.Equal(t, "3499387714231273582", result.TokenAmountOut.Amount.String())
	require.Equal(t, arbitrumUsdc, result.Fee.Token)
	require.Zero(t, result.Fee.Amount.Sign())
	require.Equal(t, int64(255_333), result.Gas)
	swap, ok := result.SwapInfo.(*math.Swap)
	require.True(t, ok)
	require.Equal(t, result.TokenAmountOut.Amount.String(), swap.Quote.AmountOut.Dec())
}

// TestCalcAmountOut_Sell: a WETH input sells into level -1, the math package's vector.
func TestCalcAmountOut_Sell(t *testing.T) {
	sim := newSim(t, simExtra(t, nowMs()), true)
	result, err := quote(sim, arbitrumWeth, wethUnits(t, 5), arbitrumUsdc)
	require.NoError(t, err)
	require.Equal(t, arbitrumUsdc, result.TokenAmountOut.Token)
	require.Equal(t, "9997501249", result.TokenAmountOut.Amount.String())
	require.Equal(t, int64(258_833), result.Gas) // Skip level 0 and write level -1.
}

func TestCalcAmountOut_GasAtLevelBoundary(t *testing.T) {
	sim := newSim(t, simExtra(t, 0), false)
	boundary := usdcUnits(t, 20_010)
	exact, err := quote(sim, arbitrumUsdc, boundary, arbitrumWeth)
	require.NoError(t, err)
	require.Equal(t, int64(255_333), exact.Gas)

	across, err := quote(sim, arbitrumUsdc, new(uint256.Int).AddUint64(boundary, 1), arbitrumWeth)
	require.NoError(t, err)
	require.Equal(t, int64(294_333), across.Gas)
}

func TestCalcAmountOut_Error(t *testing.T) {
	for _, c := range []struct {
		name              string
		change            func(*Extra)
		tokenIn, tokenOut string
		amountIn          *big.Int
		want              error
	}{
		{"unknown token", nil, "0x0000000000000000000000000000000000000001", arbitrumWeth, big.NewInt(1), ErrInvalidToken},
		{"same token", nil, arbitrumUsdc, arbitrumUsdc, big.NewInt(1), ErrInvalidToken},
		{"negative amount", nil, arbitrumUsdc, arbitrumWeth, big.NewInt(-1), ErrInvalidAmount},
		{"zero amount", nil, arbitrumUsdc, arbitrumWeth, big.NewInt(0), math.ErrZeroSwapAmount},
		{"not routed", func(e *Extra) { e.Routed = false }, arbitrumUsdc, arbitrumWeth, big.NewInt(1), ErrNoRoute},
		{"router paused", func(e *Extra) { e.RouterPaused = true }, arbitrumUsdc, arbitrumWeth, big.NewInt(1), ErrPaused},
		{"pool paused", func(e *Extra) { e.PoolPaused = true }, arbitrumUsdc, arbitrumWeth, big.NewInt(1), ErrPaused},
		{"router not swapper", func(e *Extra) { e.RouterSwapper = false }, arbitrumUsdc, arbitrumWeth, big.NewInt(1), ErrMissingRole},
		{"pool not price reader", func(e *Extra) { e.PoolPriceReader = false }, arbitrumUsdc, arbitrumWeth, big.NewInt(1), ErrMissingRole},
		{"partner without role", func(e *Extra) { e.PartnerRole = false }, arbitrumUsdc, arbitrumWeth, big.NewInt(1), ErrPartnerNotEnabled},
		{"partner without fee recipient", func(e *Extra) { e.PartnerFeeRecipientSet = false }, arbitrumUsdc, arbitrumWeth, big.NewInt(1), ErrPartnerNotEnabled},
		{"no price", func(e *Extra) { e.PriceAvailable = false }, arbitrumUsdc, arbitrumWeth, big.NewInt(1), ErrPriceUnavailable},
		// All 30 WETH cost 60,070 USDC.
		{"insufficient liquidity", nil, arbitrumUsdc, arbitrumWeth, usdcUnits(t, 60_071).ToBig(), math.ErrInsufficientLiquidity},
	} {
		t.Run(c.name, func(t *testing.T) {
			extra := simExtra(t, nowMs())
			if c.change != nil {
				c.change(&extra)
			}
			_, err := newSim(t, extra, true).CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: c.tokenIn, Amount: c.amountIn},
				TokenOut:      c.tokenOut,
			})
			require.ErrorIs(t, err, c.want)
		})
	}
}

// TestCalcAmountOut_PriceAge: with StaleCheck, the age check counts to the quote's time, so for a
// 30-second maximum age a price 29 s old is fresh and a price 31 s old is stale. Without
// StaleCheck, as the liquidity indexer runs, no price is stale.
func TestCalcAmountOut_PriceAge(t *testing.T) {
	fresh := newSim(t, simExtra(t, nowMs()-29_000), true)
	_, err := quote(fresh, arbitrumUsdc, usdcUnits(t, 1_000), arbitrumWeth)
	require.NoError(t, err)

	stale := simExtra(t, nowMs()-31_000)
	_, err = quote(newSim(t, stale, true), arbitrumUsdc, usdcUnits(t, 1_000), arbitrumWeth)
	require.ErrorIs(t, err, math.ErrStaleOraclePrice)

	old := simExtra(t, nowMs()-3_600_000)
	_, err = quote(newSim(t, old, true), arbitrumWeth, wethUnits(t, 1), arbitrumUsdc)
	require.ErrorIs(t, err, math.ErrStaleOraclePrice)
	_, err = quote(newSim(t, old, false), arbitrumWeth, wethUnits(t, 1), arbitrumUsdc)
	require.NoError(t, err)
}

// TestUpdateBalance: UpdateBalance applies the quote's swap: the next quote starts where the swap
// ended, and the reserves follow the levels.
func TestUpdateBalance(t *testing.T) {
	extra := simExtra(t, nowMs())
	sim := newSim(t, extra, true)
	result, err := quote(sim, arbitrumUsdc, usdcUnits(t, 25_000), arbitrumWeth)
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: result.SwapInfo})

	swap := result.SwapInfo.(*math.Swap)
	expected := extra.Pool
	expected.Levels = append([]math.Level(nil), extra.Pool.Levels...)
	require.NoError(t, expected.Apply(*swap))
	require.Equal(t, expected, sim.Extra.Pool)

	baseLeft := new(big.Int).Sub(wethUnits(t, 30).ToBig(), result.TokenAmountOut.Amount)
	require.Equal(t, baseLeft.String(), sim.Info.Reserves[0].String())
	require.Equal(t, usdcUnits(t, 45_000).Dec(), sim.Info.Reserves[1].String())

	next, err := quote(sim, arbitrumWeth, wethUnits(t, 1), arbitrumUsdc)
	require.NoError(t, err)
	want, err := expected.SellExactBase(wethUnits(t, 1), extra.Price, 0)
	require.NoError(t, err)
	require.Equal(t, want.Quote.AmountOut.Dec(), next.TokenAmountOut.Amount.String())
}

// TestUpdateBalance_OtherSwapInfo: a SwapInfo that is not this simulator's changes nothing.
func TestUpdateBalance_OtherSwapInfo(t *testing.T) {
	sim := newSim(t, simExtra(t, nowMs()), true)
	before := sim.Extra
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: nil})
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: math.Swap{}})
	require.Equal(t, before, sim.Extra)
}

// TestCloneState: a clone keeps its levels and position when the original is updated.
func TestCloneState(t *testing.T) {
	sim := newSim(t, simExtra(t, nowMs()), true)
	clone := sim.CloneState().(*PoolSimulator)
	before, err := quote(clone, arbitrumUsdc, usdcUnits(t, 7_000), arbitrumWeth)
	require.NoError(t, err)

	result, err := quote(sim, arbitrumUsdc, usdcUnits(t, 25_000), arbitrumWeth)
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: result.SwapInfo})

	after, err := quote(clone, arbitrumUsdc, usdcUnits(t, 7_000), arbitrumWeth)
	require.NoError(t, err)
	require.Equal(t, before.TokenAmountOut, after.TokenAmountOut)
	require.Equal(t, simExtra(t, 0).Pool.Levels, clone.Extra.Pool.Levels)
	require.Equal(t, math.Position{LevelID: 0, Fraction: math.Lower}, clone.Extra.Pool.Position)
}

// TestPoolSimulator_Msgpack: a simulator whose position is inside a level, after an update,
// round-trips through msgpack with Kyber's encoder options and quotes the same.
func TestPoolSimulator_Msgpack(t *testing.T) {
	sim := newSim(t, simExtra(t, nowMs()), true)
	result, err := quote(sim, arbitrumUsdc, usdcUnits(t, 7_000), arbitrumWeth)
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: result.SwapInfo})

	var buffer bytes.Buffer
	encoder := msgpack.NewEncoder(&buffer)
	encoder.IncludeUnexported(true)
	encoder.SetForceAsArray(true)
	require.NoError(t, encoder.Encode(sim))
	var decoded PoolSimulator
	decoder := msgpack.NewDecoder(&buffer)
	decoder.IncludeUnexported(true)
	decoder.SetForceAsArray(true)
	require.NoError(t, decoder.Decode(&decoded))

	require.Equal(t, sim.Extra, decoded.Extra)
	require.False(t, decoded.Extra.Pool.Position.Fraction.IsLower())
	require.Equal(t, sim.StaticExtra, decoded.StaticExtra)
	require.True(t, decoded.StaleCheck)
	want, err := quote(sim, arbitrumWeth, wethUnits(t, 1), arbitrumUsdc)
	require.NoError(t, err)
	got, err := quote(&decoded, arbitrumWeth, wethUnits(t, 1), arbitrumUsdc)
	require.NoError(t, err)
	require.Equal(t, want.TokenAmountOut.Amount.String(), got.TokenAmountOut.Amount.String())
}

// Kyber stores simulator snapshots as arrays, so the executor keeps its original field position.
func TestStaticExtra_MsgpackPreviousSnapshot(t *testing.T) {
	var buffer bytes.Buffer
	encoder := msgpack.NewEncoder(&buffer)
	encoder.SetForceAsArray(true)
	old := struct{ RouterAddress, PartnerAddress string }{simRouter, "0x000000000000000000000000000000000000cafe"}
	require.NoError(t, encoder.Encode(old))
	decoder := msgpack.NewDecoder(&buffer)
	decoder.SetForceAsArray(true)
	var got StaticExtra
	require.NoError(t, decoder.Decode(&got))
	require.Equal(t, StaticExtra{RouterAddress: old.RouterAddress, KyberExecutor: old.PartnerAddress}, got)
}

func TestPoolSimulator_Router(t *testing.T) {
	sim := newSim(t, simExtra(t, nowMs()), true)
	require.Equal(t, pool.MetaInfo{ApprovalAddress: simRouter, BlockNumber: 525_043_339},
		sim.GetMetaInfo(arbitrumUsdc, arbitrumWeth))
	require.Equal(t, simRouter, sim.GetApprovalAddress(arbitrumUsdc, arbitrumWeth))
}

// TestNewPoolSimulator_WithoutState: a record that GetNewPools listed has no state yet.
func TestNewPoolSimulator_WithoutState(t *testing.T) {
	_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: entity.Pool{
		Address:     arbitrumWethUsdcPool,
		Tokens:      []*entity.PoolToken{{Address: arbitrumWeth}, {Address: arbitrumUsdc}},
		Reserves:    entity.PoolReserves{"0", "0"},
		StaticExtra: `{"routerAddress":"` + simRouter + `"}`,
	}})
	require.Error(t, err)
}
