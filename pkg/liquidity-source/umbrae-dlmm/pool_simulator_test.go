package umbraedlmm

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	tokenX = "0x14a4e80d633af55ace1160c320f5a36d41cced3e" // U1
	tokenY = "0x4200000000000000000000000000000000000006" // WETH
)

func newSim(t *testing.T, static StaticExtra, extra Extra) *PoolSimulator {
	t.Helper()
	se, _ := json.Marshal(static)
	ex, _ := json.Marshal(extra)
	sim, err := NewPoolSimulator(entity.Pool{
		Address:     "0x697b72320656e6dc60db7a4bfb95084c9d9c55a0",
		Exchange:    DexType,
		Type:        DexType,
		Reserves:    entity.PoolReserves{"0", "0"},
		Tokens:      []*entity.PoolToken{{Address: tokenX}, {Address: tokenY}},
		StaticExtra: string(se),
		Extra:       string(ex),
	})
	require.NoError(t, err)
	return sim
}

func u(v string) *uint256.Int { return uint256.MustFromDecimal(v) }

// TestCalcAmountOut_SingleBin verifies the full traversal arithmetic against a hand-computed case:
// one active bin, 18/18 decimals, binStep 25, base fee 30 bps, variable fee disabled.
func TestCalcAmountOut_SingleBin(t *testing.T) {
	static := StaticExtra{BinStep: 25, DecimalsX: 18, DecimalsY: 18}
	extra := Extra{
		ActiveID: activeBinID,
		Bins: []Bin{
			{ID: activeBinID, ReserveX: u("0"), ReserveY: u("1000000000000000000000")}, // 1000 Y
		},
		FeeParameters: FeeParameters{
			BaseFactor: 30, VariableFeeControl: 0,
			MaxVolatilityAccumulator: 35000, MinSwapBps: 0,
		},
		NativeReserveX: "0",
		NativeReserveY: "1000000000000000000000",
	}
	sim := newSim(t, static, extra)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: big.NewInt(1_000_000_000_000_000_000)}, // 1 X
		TokenOut:      tokenY,
	})
	require.NoError(t, err)
	// inAfterFee = 1e18 - floor(1e18*30/10030); at a 1:1 bin price the full inAfterFee converts.
	require.Equal(t, "997008973080757727", res.TokenAmountOut.Amount.String())
	require.Equal(t, "2991026919242273", res.Fee.Amount.String())
}

func TestCalcAmountOut_UpdateBalanceAndClone(t *testing.T) {
	static := StaticExtra{BinStep: 25, DecimalsX: 18, DecimalsY: 18}
	extra := Extra{
		ActiveID: activeBinID,
		Bins: []Bin{
			{ID: activeBinID, ReserveX: u("0"), ReserveY: u("1000000000000000000000")},
		},
		FeeParameters:  FeeParameters{BaseFactor: 30, VariableFeeControl: 0, MaxVolatilityAccumulator: 35000},
		NativeReserveX: "0",
		NativeReserveY: "1000000000000000000000",
	}
	sim := newSim(t, static, extra)
	in := big.NewInt(1_000_000_000_000_000_000)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: in}, TokenOut: tokenY,
	})
	require.NoError(t, err)

	clone := sim.CloneState()
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: tokenX, Amount: in},
		TokenAmountOut: pool.TokenAmount{Token: tokenY, Amount: res.TokenAmountOut.Amount},
		Fee:            *res.Fee,
		SwapInfo:       res.SwapInfo,
	})

	// Active bin Y reserve dropped by the output amount.
	wantY := new(uint256.Int).Sub(u("1000000000000000000000"), uint256.MustFromBig(res.TokenAmountOut.Amount))
	require.Equal(t, wantY, sim.bins[0].ReserveY)
	// Clone is untouched.
	require.Equal(t, u("1000000000000000000000"), clone.(*PoolSimulator).bins[0].ReserveY)
}

// TestCalcAmountOut_DecimalScaling exercises the native<->normalized conversion (scaleIn/scaleOut/
// precisionX) and the decimalsY price adjustment on a synthetic 6-dec X / 18-dec Y pool — the path
// no on-chain Umbrae pair covers yet (all live pairs are 18/18). Values are hand-computed at the
// active (1:1 normalized) bin, base fee 30 bps, variable fee off. The fee is charged in input-token
// native decimals, so the 6-dec direction rounds coarser than 18/18 — that mirrors the contract.
func TestCalcAmountOut_DecimalScaling(t *testing.T) {
	// scaleX = 10^(18-6) = 1e12, precisionX = 10^6, scaleY = 1.
	static := StaticExtra{BinStep: 25, DecimalsX: 6, DecimalsY: 18}
	extra := Extra{
		ActiveID: activeBinID,
		Bins: []Bin{
			// Normalized reserves: 1000 of each token (X: 1e9 native * 1e12 scale = 1e21; Y: 1e21).
			{ID: activeBinID, ReserveX: u("1000000000000000000000"), ReserveY: u("1000000000000000000000")},
		},
		FeeParameters:  FeeParameters{BaseFactor: 30, VariableFeeControl: 0, MaxVolatilityAccumulator: 35000, MinSwapBps: 0},
		NativeReserveX: "1000000000", NativeReserveY: "1000000000000000000000",
	}
	sim := newSim(t, static, extra)

	// X->Y: 1 whole X = 1e6 native in. Fee (native X) = floor(1e6*30/10030) = 2991.
	resXY, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: big.NewInt(1_000_000)}, TokenOut: tokenY,
	})
	require.NoError(t, err)
	require.Equal(t, "997009000000000000", resXY.TokenAmountOut.Amount.String(), "X->Y out (Y, 18-dec)")
	require.Equal(t, "2991", resXY.Fee.Amount.String(), "X->Y fee (X, 6-dec)")

	// Y->X: 1 Y = 1e18 native in. Fee (native Y) = floor(1e18*30/10030); out floors to 6-dec X.
	resYX, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenY, Amount: big.NewInt(1_000_000_000_000_000_000)}, TokenOut: tokenX,
	})
	require.NoError(t, err)
	require.Equal(t, "997008", resYX.TokenAmountOut.Amount.String(), "Y->X out (X, 6-dec)")
	require.Equal(t, "2991026919242273", resYX.Fee.Amount.String(), "Y->X fee (Y, 18-dec)")
}

func TestCalcAmountOut_InvalidToken(t *testing.T) {
	sim := newSim(t, StaticExtra{BinStep: 25, DecimalsX: 18, DecimalsY: 18}, Extra{
		ActiveID:       activeBinID,
		Bins:           []Bin{{ID: activeBinID, ReserveX: u("0"), ReserveY: u("1000000000000000000000")}},
		FeeParameters:  FeeParameters{BaseFactor: 30, MaxVolatilityAccumulator: 35000},
		NativeReserveX: "0", NativeReserveY: "1000000000000000000000",
	})
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xdead", Amount: big.NewInt(1)}, TokenOut: tokenY,
	})
	require.ErrorIs(t, err, ErrInvalidToken)
}
