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
	tokenX = "0x4200000000000000000000000000000000000006" // WETH (tokenX = lower address)
	tokenY = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913" // USDC
)

func newSim(t *testing.T, static StaticExtra, extra Extra) *PoolSimulator {
	t.Helper()
	se, _ := json.Marshal(static)
	ex, _ := json.Marshal(extra)
	sim, err := NewPoolSimulator(entity.Pool{
		Address:     "0x4a27eeff8abe2a467425fdac65e5b579e26a90b5",
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

// baseFeeParams keeps the variable term off (variableFeeControl=0) so hand-computed fixtures only
// exercise the base fee.
func baseFeeParams() FeeParameters {
	return FeeParameters{BaseFactor: 30, VariableFeeControl: 0, MaxVolatilityAccumulator: 35000}
}

// TestCalcAmountOut_SingleBin verifies the full traversal arithmetic against a hand-computed case:
// one active bin, 18/18 decimals, binStep 25, base fee 30 bps, variable fee disabled.
// V2 fee CEILS: fee = ceil(1e18*30/10030) = 2991026919242274 (V1 floored to ...273).
func TestCalcAmountOut_SingleBin(t *testing.T) {
	static := StaticExtra{BinStep: 25, DecimalsX: 18, DecimalsY: 18}
	extra := Extra{
		ActiveID: activeBinID,
		Bins: []Bin{
			{ID: activeBinID, ReserveX: u("0"), ReserveY: u("1000000000000000000000")}, // 1000 Y
		},
		FeeParameters: baseFeeParams(),
	}
	sim := newSim(t, static, extra)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: big.NewInt(1_000_000_000_000_000_000)}, // 1 X
		TokenOut:      tokenY,
	})
	require.NoError(t, err)
	// inAfterFee = 1e18 - 2991026919242274; at a 1:1 bin price the full inAfterFee converts.
	require.Equal(t, "997008973080757726", res.TokenAmountOut.Amount.String())
	require.Equal(t, "2991026919242274", res.Fee.Amount.String())
}

func TestCalcAmountOut_UpdateBalanceAndClone(t *testing.T) {
	static := StaticExtra{BinStep: 25, DecimalsX: 18, DecimalsY: 18}
	extra := Extra{
		ActiveID: activeBinID,
		Bins: []Bin{
			{ID: activeBinID, ReserveX: u("0"), ReserveY: u("1000000000000000000000")},
		},
		FeeParameters: baseFeeParams(),
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

// TestCalcAmountOut_DecimalScaling exercises the native<->normalized conversion and the V2 unified
// price domain (priceDenominator = scaleY * 10^decimalsX, identical in both directions) on an
// 18-dec X / 6-dec Y pool — the live WETH/USDC shape. Values are hand-computed at the active
// (1:1 normalized) bin, base fee 30 bps, variable fee off. The fee is charged in input-token
// native decimals and CEILS, so the 6-dec direction rounds coarser — that mirrors the contract.
func TestCalcAmountOut_DecimalScaling(t *testing.T) {
	// scaleX = 1, scaleY = 10^(18-6) = 1e12, priceDenominator = 1e12 * 1e18 = 1e30.
	static := StaticExtra{BinStep: 25, DecimalsX: 18, DecimalsY: 6}
	extra := Extra{
		ActiveID: activeBinID,
		Bins: []Bin{
			// Normalized reserves: 1000 of each token (X: 1e21; Y: 1e9 native * 1e12 scale = 1e21).
			{ID: activeBinID, ReserveX: u("1000000000000000000000"), ReserveY: u("1000000000000000000000")},
		},
		FeeParameters: baseFeeParams(),
	}
	sim := newSim(t, static, extra)

	// X->Y: 1 whole X = 1e18 native in. Fee (native X) = ceil(1e18*30/10030) = 2991026919242274.
	// net = 997008973080757726; out native Y = floor(net * 1e18 / 1e30) = 997008 (6-dec).
	resXY, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: big.NewInt(1_000_000_000_000_000_000)}, TokenOut: tokenY,
	})
	require.NoError(t, err)
	require.Equal(t, "997008", resXY.TokenAmountOut.Amount.String(), "X->Y out (Y, 6-dec)")
	require.Equal(t, "2991026919242274", resXY.Fee.Amount.String(), "X->Y fee (X, 18-dec)")

	// Y->X: 1 whole Y = 1e6 native in. Fee (native Y) = ceil(1e6*30/10030) = 2992 (V1: 2991).
	// net = 997008; out native X = floor(997008 * 1e30 / 1e18 / 1) = 997008e12.
	resYX, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenY, Amount: big.NewInt(1_000_000)}, TokenOut: tokenX,
	})
	require.NoError(t, err)
	require.Equal(t, "997008000000000000", resYX.TokenAmountOut.Amount.String(), "Y->X out (X, 18-dec)")
	require.Equal(t, "2992", resYX.Fee.Amount.String(), "Y->X fee (Y, 6-dec)")
}

// TestCalcAmountOut_InsufficientLiquidity covers the case that reverts on-chain: the input requires
// more output than the whole book holds. Because the router-service always sends the full amountIn,
// a partial fill would leave the route reverting on-chain (LBPair__InsufficientLiquidityForSwap),
// so the simulator must return ErrInsufficientLiquidity instead of a partial quote.
func TestCalcAmountOut_InsufficientLiquidity(t *testing.T) {
	static := StaticExtra{BinStep: 25, DecimalsX: 18, DecimalsY: 18}
	// Single active bin with only 1000 Y. A huge X input cannot be fully consumed.
	extra := Extra{
		ActiveID:      activeBinID,
		Bins:          []Bin{{ID: activeBinID, ReserveX: u("0"), ReserveY: u("1000000000000000000000")}},
		FeeParameters: baseFeeParams(),
	}
	sim := newSim(t, static, extra)

	in := new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil) // 1e24 X, far more than the bin can absorb
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: in}, TokenOut: tokenY,
	})
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
}

// TestCalcAmountOut_MovementBound covers the V2 movement bound (SwapMovement.isAllowed, #278): a
// swap may not reach a bin further than distance*binStep = 10000 bps from the entry bin — at
// binStep 25 that is 400 bins. Execution reverts there, so the simulator must refuse the quote.
func TestCalcAmountOut_MovementBound(t *testing.T) {
	static := StaticExtra{BinStep: 25, DecimalsX: 18, DecimalsY: 18}
	mk := func(farOffset uint32) *PoolSimulator {
		return newSim(t, static, Extra{
			ActiveID: activeBinID,
			Bins: []Bin{
				{ID: activeBinID, ReserveX: u("0"), ReserveY: u("1000000000000000000")},          // 1 Y
				{ID: activeBinID - farOffset, ReserveX: u("0"), ReserveY: u("1000000000000000000000")}, // 1000 Y
			},
			FeeParameters: baseFeeParams(),
		})
	}
	in := big.NewInt(2_000_000_000_000_000_000) // 2 X — drains the active bin, needs the far bin

	// Far bin at distance 400 (400*25 = 10000 bps): allowed.
	res, err := mk(400).CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: in}, TokenOut: tokenY,
	})
	require.NoError(t, err)
	require.Positive(t, res.TokenAmountOut.Amount.Sign())

	// Far bin at distance 401 (401*25 > 10000 bps): the walk must refuse, as execution reverts.
	_, err = mk(401).CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: in}, TokenOut: tokenY,
	})
	require.ErrorIs(t, err, ErrSwapMovementExceeded)
}

// TestCalcAmountOut_FullyConsumable ensures a swap the book can fully absorb succeeds (no error).
func TestCalcAmountOut_FullyConsumable(t *testing.T) {
	static := StaticExtra{BinStep: 25, DecimalsX: 18, DecimalsY: 18}
	extra := Extra{
		ActiveID:      activeBinID,
		Bins:          []Bin{{ID: activeBinID, ReserveX: u("0"), ReserveY: u("1000000000000000000000")}},
		FeeParameters: baseFeeParams(),
	}
	sim := newSim(t, static, extra)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: big.NewInt(1_000_000_000_000_000_000)}, TokenOut: tokenY,
	})
	require.NoError(t, err)
	require.Positive(t, res.TokenAmountOut.Amount.Sign())
}

func TestCalcAmountOut_InvalidToken(t *testing.T) {
	sim := newSim(t, StaticExtra{BinStep: 25, DecimalsX: 18, DecimalsY: 18}, Extra{
		ActiveID:      activeBinID,
		Bins:          []Bin{{ID: activeBinID, ReserveX: u("0"), ReserveY: u("1000000000000000000000")}},
		FeeParameters: baseFeeParams(),
	})
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xdead", Amount: big.NewInt(1)}, TokenOut: tokenY,
	})
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestGetMetaInfo_ApprovalAddressMatchesRouter(t *testing.T) {
	router := "0x5a7673d413f510f4b5c191d96837694c5942bf38" // V2 DLMM Router
	sim := newSim(t, StaticExtra{BinStep: 25, DecimalsX: 18, DecimalsY: 18, Router: router}, Extra{
		ActiveID:      activeBinID,
		Bins:          []Bin{{ID: activeBinID, ReserveX: u("0"), ReserveY: u("1000000000000000000000")}},
		FeeParameters: baseFeeParams(),
	})
	meta, ok := sim.GetMetaInfo(tokenX, tokenY).(PoolMeta)
	require.True(t, ok)
	require.Equal(t, router, meta.ApprovalAddress)
	require.Equal(t, sim.GetApprovalAddress(tokenX, tokenY), meta.ApprovalAddress)
	require.Equal(t, uint16(25), meta.BinStep)
}
