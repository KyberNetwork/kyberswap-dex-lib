package stake

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	wbnbAddr    = "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"
	slisBnbAddr = "0xb0b84d294e0c75a6abe60171b70edeb2efd14a1b"
)

func mustUint256(s string) *uint256.Int {
	return uint256.MustFromDecimal(s)
}

func buildPool(t *testing.T, extra Extra, buffer string) entity.Pool {
	t.Helper()
	return buildPoolWithStaticExtra(t, extra, buffer, StaticExtra{IsNativeUnderlying: true})
}

func buildPoolWithStaticExtra(t *testing.T, extra Extra, buffer string, staticExtra StaticExtra) entity.Pool {
	t.Helper()
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)
	staticExtraBytes, err := json.Marshal(staticExtra)
	require.NoError(t, err)

	return entity.Pool{
		Address:     "0x1adb950d8bb3da4be104211d5ab038628e477fe6",
		Exchange:    "lista-stake",
		Type:        DexType,
		Reserves:    []string{buffer, defaultDepositReserve},
		StaticExtra: string(staticExtraBytes),
		Tokens: []*entity.PoolToken{
			{Address: wbnbAddr, Swappable: true},
			{Address: slisBnbAddr, Swappable: true},
		},
		Extra: string(extraBytes),
	}
}

func TestCalcAmountOut_Deposit(t *testing.T) {
	t.Parallel()

	extra := Extra{
		Paused:                  false,
		DepositRate:             mustUint256("980392156862745098"),  // convertBnbToSnBnb(1e18)
		WithdrawRate:            mustUint256("1020000000000000000"), // convertSnBnbToBnb(1e18)
		InstantWithdrawFeeRate:  mustUint256("0"),
		MinBnb:                  mustUint256("1000000000000000"),
		InstantWithdrawEligible: false,
	}
	sim, err := NewPoolSimulator(buildPool(t, extra, "2000000000000000000"))
	require.NoError(t, err)

	// CanSwapFrom(WBNB) should only allow -> slisBNB when not eligible for instant withdraw.
	assert.Equal(t, []string{slisBnbAddr}, sim.CanSwapFrom(wbnbAddr))
	assert.Empty(t, sim.CanSwapFrom(slisBnbAddr))

	res, err := sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: wbnbAddr, Amount: bignumber.NewBig("1000000000000000000")},
		TokenOut:      slisBnbAddr,
	})
	require.NoError(t, err)
	assert.Equal(t, "980392156862745098", res.TokenAmountOut.Amount.String())
}

func TestCalcAmountOut_InstantWithdraw_NotEligible(t *testing.T) {
	t.Parallel()

	extra := Extra{
		DepositRate:             mustUint256("980392156862745098"),
		WithdrawRate:            mustUint256("1020000000000000000"),
		InstantWithdrawFeeRate:  mustUint256("0"),
		MinBnb:                  mustUint256("1000000000000000"),
		InstantWithdrawEligible: false,
	}
	sim, err := NewPoolSimulator(buildPool(t, extra, "2000000000000000000"))
	require.NoError(t, err)

	assert.Empty(t, sim.CanSwapFrom(slisBnbAddr))

	_, err = sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: slisBnbAddr, Amount: bignumber.NewBig("1000000000000000000")},
		TokenOut:      wbnbAddr,
	})
	assert.ErrorIs(t, err, ErrInstantWithdrawNotEligible)
}

func TestCalcAmountOut_InstantWithdraw_Eligible(t *testing.T) {
	t.Parallel()

	// 1% instant withdraw fee.
	extra := Extra{
		DepositRate:             mustUint256("980392156862745098"),
		WithdrawRate:            mustUint256("1020000000000000000"),
		InstantWithdrawFeeRate:  mustUint256("100000000"), // 1e8 / 1e10 = 1%
		MinBnb:                  mustUint256("1000000000000000"),
		InstantWithdrawEligible: true,
	}
	sim, err := NewPoolSimulator(buildPool(t, extra, "2000000000000000000")) // 2 BNB buffer
	require.NoError(t, err)

	assert.Equal(t, []string{wbnbAddr}, sim.CanSwapFrom(slisBnbAddr))

	// 1 slisBNB in: fee = 0.01e18, burn = 0.99e18, out = 0.99e18 * 1.02e18 / 1e18 = 1009800000000000000
	res, err := sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: slisBnbAddr, Amount: bignumber.NewBig("1000000000000000000")},
		TokenOut:      wbnbAddr,
	})
	require.NoError(t, err)
	assert.Equal(t, "1009800000000000000", res.TokenAmountOut.Amount.String())

	// Exceeds the 2 BNB buffer -> insufficient liquidity.
	_, err = sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: slisBnbAddr, Amount: bignumber.NewBig("5000000000000000000")},
		TokenOut:      wbnbAddr,
	})
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)

	// Below minBnb -> amount too small.
	_, err = sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: slisBnbAddr, Amount: bignumber.NewBig("1000000000000")},
		TokenOut:      wbnbAddr,
	})
	assert.ErrorIs(t, err, ErrAmountTooSmall)
}

// A hypothetical variant configured with IsNativeUnderlying=false (deposit takes the token via
// transferFrom, not payable) must report no native support even though it trades the exact same
// wrapped token address as the real, native-underlying instance -- the flag, not the address, is
// what SwapReceiveNativeIn/SwapReturnNativeOut key off.
func TestSwapNativeInOut_TracksConfiguredFlagNotTokenAddress(t *testing.T) {
	t.Parallel()

	extra := Extra{
		DepositRate:             mustUint256("980392156862745098"),
		WithdrawRate:            mustUint256("1020000000000000000"),
		InstantWithdrawFeeRate:  mustUint256("0"),
		MinBnb:                  mustUint256("1000000000000000"),
		InstantWithdrawEligible: true,
	}

	nativeSim, err := NewPoolSimulator(
		buildPoolWithStaticExtra(t, extra, "2000000000000000000", StaticExtra{IsNativeUnderlying: true}),
	)
	require.NoError(t, err)
	assert.True(t, nativeSim.SwapReceiveNativeIn(wbnbAddr, slisBnbAddr, valueobject.ChainIDBSC))
	assert.True(t, nativeSim.SwapReturnNativeOut(slisBnbAddr, wbnbAddr, valueobject.ChainIDBSC))

	wrappedOnlySim, err := NewPoolSimulator(
		buildPoolWithStaticExtra(t, extra, "2000000000000000000", StaticExtra{IsNativeUnderlying: false}),
	)
	require.NoError(t, err)
	assert.False(t, wrappedOnlySim.SwapReceiveNativeIn(wbnbAddr, slisBnbAddr, valueobject.ChainIDBSC))
	assert.False(t, wrappedOnlySim.SwapReturnNativeOut(slisBnbAddr, wbnbAddr, valueobject.ChainIDBSC))

	// Neither ever reports native support on the wrong side of the swap (slisBNB is never native).
	assert.False(t, nativeSim.SwapReceiveNativeIn(slisBnbAddr, wbnbAddr, valueobject.ChainIDBSC))
	assert.False(t, nativeSim.SwapReturnNativeOut(wbnbAddr, slisBnbAddr, valueobject.ChainIDBSC))
}

// GetMetaInfo's IsWithdraw must key off token identity alone, not nativeness -- it has to stay
// correct even for a hypothetical instance whose underlying is neither native nor wrapped-native
// (e.g. some other plain ERC20), which is exactly the case a naive IsNative/IsWrappedNative(tokenIn)
// check in the encoder got wrong.
func TestGetMetaInfo(t *testing.T) {
	t.Parallel()

	extra := Extra{
		DepositRate:             mustUint256("980392156862745098"),
		WithdrawRate:            mustUint256("1020000000000000000"),
		InstantWithdrawFeeRate:  mustUint256("0"),
		MinBnb:                  mustUint256("1000000000000000"),
		InstantWithdrawEligible: true,
	}

	nonNativeSim, err := NewPoolSimulator(
		buildPoolWithStaticExtra(t, extra, "2000000000000000000", StaticExtra{IsNativeUnderlying: false}),
	)
	require.NoError(t, err)
	assert.Equal(t, Meta{IsWithdraw: false, IsNativeUnderlying: false}, nonNativeSim.GetMetaInfo(wbnbAddr, slisBnbAddr))
	assert.Equal(t, Meta{IsWithdraw: true, IsNativeUnderlying: false}, nonNativeSim.GetMetaInfo(slisBnbAddr, wbnbAddr))

	nativeSim, err := NewPoolSimulator(
		buildPoolWithStaticExtra(t, extra, "2000000000000000000", StaticExtra{IsNativeUnderlying: true}),
	)
	require.NoError(t, err)
	assert.Equal(t, Meta{IsWithdraw: false, IsNativeUnderlying: true}, nativeSim.GetMetaInfo(wbnbAddr, slisBnbAddr))
	assert.Equal(t, Meta{IsWithdraw: true, IsNativeUnderlying: true}, nativeSim.GetMetaInfo(slisBnbAddr, wbnbAddr))
}

func TestCalcAmountOut_Paused(t *testing.T) {
	t.Parallel()

	extra := Extra{
		Paused:                  true,
		DepositRate:             mustUint256("980392156862745098"),
		WithdrawRate:            mustUint256("1020000000000000000"),
		InstantWithdrawFeeRate:  mustUint256("0"),
		MinBnb:                  mustUint256("1000000000000000"),
		InstantWithdrawEligible: true,
	}
	sim, err := NewPoolSimulator(buildPool(t, extra, "2000000000000000000"))
	require.NoError(t, err)

	assert.Empty(t, sim.CanSwapFrom(wbnbAddr))
	assert.Empty(t, sim.CanSwapFrom(slisBnbAddr))

	_, err = sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: wbnbAddr, Amount: bignumber.NewBig("1000000000000000000")},
		TokenOut:      slisBnbAddr,
	})
	assert.ErrorIs(t, err, ErrPoolPaused)
}
