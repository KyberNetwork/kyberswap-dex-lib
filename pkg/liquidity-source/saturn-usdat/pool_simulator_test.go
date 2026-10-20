package saturnusdat

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func newTestPool(t *testing.T, mutate func(*Extra)) *PoolSimulator {
	t.Helper()
	extra := Extra{
		AssetCap:          u("10000000000000"),
		AssetBalance:      u("0"),
		AssetDecimals:     6,
		ExtensionDecimals: 6,
		AssetAllowed:      true,
		ExtensionApproved: true,
		WhitelistEnabled:  true,
	}
	if mutate != nil {
		mutate(&extra)
	}
	return mustPool(t, extra)
}

func mustPool(t *testing.T, extra Extra) *PoolSimulator {
	t.Helper()
	extraBytes, err := jsonMarshal(extra)
	require.NoError(t, err)
	reserves := Reserves(&extra)
	sim, err := NewPoolSimulator(entity.Pool{
		Address:     USDat,
		Exchange:    DexType,
		Type:        DexType,
		Reserves:    []string{reserves[0].String(), reserves[1].String()},
		Tokens:      []*entity.PoolToken{{Address: USDC, Decimals: 6, Swappable: true}, {Address: USDat, Decimals: 6, Swappable: true}},
		Extra:       string(extraBytes),
		StaticExtra: `{"facility":"` + Facility + `","asset":"` + USDC + `"}`,
		BlockNumber: 26197865,
	})
	require.NoError(t, err)
	return sim
}

func quote(sim *PoolSimulator, tokenIn, tokenOut, amount string) (*pool.CalcAmountOutResult, error) {
	a, _ := new(big.Int).SetString(amount, 10)
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: a},
		TokenOut:      tokenOut,
	})
}

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	sim := newTestPool(t, nil)

	res, err := quote(sim, USDC, USDat, "1000000000")
	require.NoError(t, err)
	assert.Equal(t, "1000000000", res.TokenAmountOut.Amount.String())
	assert.Equal(t, USDat, res.TokenAmountOut.Token)
	assert.Equal(t, "0", res.Fee.Amount.String())
	assert.Nil(t, res.RemainingTokenAmountIn)
	assert.EqualValues(t, 185000, res.Gas)
	assert.True(t, res.IsValid())

	// reverse direction is not quotable
	_, err = quote(sim, USDat, USDC, "1000000")
	assert.ErrorIs(t, err, ErrInvalidSwapPath)
	_, err = quote(sim, USDC, USDC, "1000000")
	assert.ErrorIs(t, err, ErrInvalidSwapPath)

	_, err = quote(sim, USDC, USDat, "0")
	assert.ErrorIs(t, err, ErrZeroAmount)
	_, err = quote(sim, USDC, USDat, "-1")
	assert.ErrorIs(t, err, ErrZeroAmount)
	huge := new(big.Int).Lsh(big.NewInt(1), 256)
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: USDC, Amount: huge}, TokenOut: USDat})
	assert.ErrorIs(t, err, ErrOverflow)
}

func TestCalcAmountOutCapEdges(t *testing.T) {
	t.Parallel()
	sim := newTestPool(t, nil)
	_, err := quote(sim, USDC, USDat, "10000000000000")
	require.NoError(t, err)
	_, err = quote(sim, USDC, USDat, "10000000000001")
	assert.ErrorIs(t, err, ErrAssetCapReached)

	// partially used capacity
	sim = newTestPool(t, func(e *Extra) { e.AssetBalance = u("9999999999000") })
	_, err = quote(sim, USDC, USDat, "1000")
	require.NoError(t, err)
	_, err = quote(sim, USDC, USDat, "1001")
	assert.ErrorIs(t, err, ErrAssetCapReached)
}

func TestCalcAmountOutFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*Extra)
		err    error
	}{
		{"facility paused", func(e *Extra) { e.FacilityPaused = true }, ErrFacilityPaused},
		{"usdat paused", func(e *Extra) { e.USDatPaused = true }, ErrUSDatPaused},
		{"extension not approved", func(e *Extra) { e.ExtensionApproved = false }, ErrExtensionNotApproved},
		{"asset not allowed", func(e *Extra) { e.AssetAllowed = false }, ErrAssetNotAllowed},
		{"whitelist disabled is fine", func(e *Extra) { e.WhitelistEnabled = false }, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := newTestPool(t, tt.mutate)
			_, err := quote(sim, USDC, USDat, "1000000")
			if tt.err == nil {
				require.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.err)
			assert.Equal(t, "0", sim.Info.Reserves[0].String())
		})
	}
}

func TestCalcAmountOutDust(t *testing.T) {
	t.Parallel()
	sim := newTestPool(t, func(e *Extra) {
		e.AssetDecimals = 18
		e.AssetCap = u("10000000000000000000000")
	})
	res, err := quote(sim, USDC, USDat, "1999999999999")
	require.NoError(t, err)
	assert.Equal(t, "1", res.TokenAmountOut.Amount.String())
	require.NotNil(t, res.RemainingTokenAmountIn)
	assert.Equal(t, "999999999999", res.RemainingTokenAmountIn.Amount.String())

	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
	assert.Equal(t, "1000000000000", sim.AssetBalance.Dec())
}

func TestUpdateBalanceConsumesCapacity(t *testing.T) {
	t.Parallel()
	sim := newTestPool(t, func(e *Extra) { e.AssetCap = u("3000000") })

	for range 3 {
		res, err := quote(sim, USDC, USDat, "1000000")
		require.NoError(t, err)
		sim.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: USDC, Amount: big.NewInt(1000000)},
			TokenAmountOut: *res.TokenAmountOut,
			SwapInfo:       res.SwapInfo,
		})
	}
	assert.Equal(t, "3000000", sim.AssetBalance.Dec())
	assert.Equal(t, "0", sim.Info.Reserves[0].String())
	assert.Equal(t, "0", sim.Info.Reserves[1].String())
	_, err := quote(sim, USDC, USDat, "1")
	assert.ErrorIs(t, err, ErrAssetCapReached)

	// missing / foreign swap info is ignored
	sim.UpdateBalance(pool.UpdateBalanceParams{})
	assert.Equal(t, "3000000", sim.AssetBalance.Dec())
}

func TestCloneState(t *testing.T) {
	t.Parallel()
	sim := newTestPool(t, nil)
	cloned := sim.CloneState().(*PoolSimulator)

	res, err := quote(sim, USDC, USDat, "5000000")
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})

	assert.Equal(t, "5000000", sim.AssetBalance.Dec())
	assert.Equal(t, "0", cloned.AssetBalance.Dec())
	assert.Equal(t, "10000000000000", cloned.Info.Reserves[0].String())
	assert.Equal(t, "9999995000000", sim.Info.Reserves[0].String())

	// CalcAmountOut does not mutate state
	_, _ = quote(cloned, USDC, USDat, "1")
	assert.Equal(t, "0", cloned.AssetBalance.Dec())
}

func TestCanSwap(t *testing.T) {
	t.Parallel()
	sim := newTestPool(t, nil)
	assert.Equal(t, []string{USDat}, sim.CanSwapFrom(USDC))
	assert.Equal(t, []string{}, sim.CanSwapFrom(USDat))
	assert.Equal(t, []string{USDC}, sim.CanSwapTo(USDat))
	assert.Equal(t, []string{}, sim.CanSwapTo(USDC))
}

func TestMetaAndReserves(t *testing.T) {
	t.Parallel()
	sim := newTestPool(t, func(e *Extra) { e.AssetBalance = u("4000000000000") })
	meta := sim.GetMetaInfo(USDC, USDat).(PoolMeta)
	assert.Equal(t, Facility, meta.ApprovalAddress)
	assert.EqualValues(t, 26197865, meta.BlockNumber)
	assert.Equal(t, "6000000000000", sim.Info.Reserves[0].String())
	assert.Equal(t, "6000000000000", sim.Info.Reserves[1].String())

	// balance above cap saturates at zero
	sim = newTestPool(t, func(e *Extra) { e.AssetBalance = u("20000000000000") })
	assert.Equal(t, "0", sim.Info.Reserves[0].String())

	// scaling overflow zeroes the extension-side reserve
	e := Extra{AssetCap: u("10"), AssetBalance: new(uint256.Int), AssetDecimals: 0, ExtensionDecimals: 100,
		AssetAllowed: true, ExtensionApproved: true}
	assert.Equal(t, "0", Reserves(&e)[1].String())
}

func TestNewPoolSimulatorInvalid(t *testing.T) {
	t.Parallel()
	_, err := NewPoolSimulator(entity.Pool{Extra: `{}`, Tokens: []*entity.PoolToken{{}, {}}})
	assert.ErrorIs(t, err, ErrInvalidExtra)
	_, err = NewPoolSimulator(entity.Pool{Extra: `not json`})
	assert.Error(t, err)
	_, err = NewPoolSimulator(entity.Pool{Extra: `{"cap":"1","bal":"0"}`, StaticExtra: `x`, Tokens: []*entity.PoolToken{{}, {}}})
	assert.Error(t, err)
}

func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }

func TestCalcAmountOutNotWhitelisted(t *testing.T) {
	t.Parallel()
	sim := newTestPool(t, func(e *Extra) { e.NotWhitelisted = true })
	_, err := quote(sim, USDC, USDat, "1000000")
	assert.ErrorIs(t, err, ErrNotWhitelisted)
	assert.Equal(t, "0", sim.Info.Reserves[0].String())
	assert.Equal(t, "0", sim.Info.Reserves[1].String())
}
