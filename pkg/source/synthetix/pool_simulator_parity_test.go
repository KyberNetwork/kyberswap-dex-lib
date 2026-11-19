package synthetix

import (
	"math/big"
	"strconv"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// newParitySim builds the Ethereum pool from synthetixEthereumExtra. Ethereum is atomic, so GetAtomicVolume
// reads the swap limit. Only Extra feeds the simulator.
func newParitySim(t *testing.T) *PoolSimulator {
	t.Helper()

	sim, err := NewPoolSimulator(entity.Pool{
		Address:  "0x08f30ecf2c15a783083ab9d5b9211c22388d0564",
		Exchange: "synthetix",
		Type:     "synthetix",
		Extra:    synthetixEthereumExtra,
	}, valueobject.ChainIDEthereum)
	require.NoError(t, err)
	return sim
}

const (
	parityTokenIn  = "0x5e74c9036fb86bd7ecdcb084a0673efc32ea31cb"
	parityTokenOut = "0xfe18be6b3bd88a2d2a7f928d00292e7a9963cfc6"
)

// TestCalcAmtOutMatchesLegacy checks native CalcAmtOut against legacy CalcAmountOut on amount out, fee, gas, and
// error. The zero-limit case binds on the volume check, so both paths must reject it.
func TestCalcAmtOutMatchesLegacy(t *testing.T) {
	t.Parallel()

	amountIn := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	parityVolume, err := newParitySim(t).GetAtomicVolume(poolPkg.TokenAmount{Token: parityTokenIn, Amount: amountIn}, parityTokenOut)
	require.NoError(t, err)
	require.NotNil(t, parityVolume)

	cases := []struct {
		name    string
		limit   *big.Int
		wantErr error
	}{
		{"limit clears volume", nil, nil},
		{"limit binds on volume", big.NewInt(0), ErrSurpassedVolumeLimit},
		{"limit equals volume", parityVolume, nil},
		{"limit one wei below volume", new(big.Int).Sub(parityVolume, big.NewInt(1)), ErrSurpassedVolumeLimit},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sim := newParitySim(t)
			key := strconv.FormatUint(sim.poolState.BlockTimestamp, 10)
			limitVal := tc.limit
			if limitVal == nil {
				limitVal = new(big.Int).Set(sim.poolState.AtomicMaxVolumePerBlock)
			}
			inventory := swaplimit.NewInventory("", swaplimit.ToU256Balances(map[string]*big.Int{key: limitVal}))

			legacy, legacyErr := sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
				TokenAmountIn: poolPkg.TokenAmount{Token: parityTokenIn, Amount: amountIn},
				TokenOut:      parityTokenOut,
				Limit:         poolPkg.AsSwapLimitBig(inventory),
			})
			native, nativeErr := sim.CalcAmtOut(poolPkg.CalcAmtOutParams{
				TokenAmountIn: poolPkg.TokenAmt{Token: parityTokenIn, Amount: *uint256.MustFromBig(amountIn)},
				TokenOut:      parityTokenOut,
				Limit:         inventory,
			})

			require.ErrorIs(t, legacyErr, tc.wantErr)
			require.ErrorIs(t, nativeErr, tc.wantErr)
			if tc.wantErr != nil {
				return
			}
			require.Equal(t, legacy.TokenAmountOut.Amount.String(), native.TokenAmountOut.Amount.Dec())
			require.Equal(t, legacy.Fee.Amount.String(), native.Fee.Amount.Dec())
			require.Equal(t, legacy.Gas, native.Gas)
		})
	}
}

// TestCalcAmtOutPinnedEthereum anchors native CalcAmtOut to the amount, fee, and gas the existing Ethereum case
// expects. Parity alone cannot catch a change to the shared core, since legacy forwards native.
func TestCalcAmtOutPinnedEthereum(t *testing.T) {
	t.Parallel()

	sim := newParitySim(t)
	inventory := swaplimit.NewInventory("", swaplimit.ToU256Balances(map[string]*big.Int{
		strconv.FormatUint(sim.poolState.BlockTimestamp, 10): new(big.Int).Set(sim.poolState.AtomicMaxVolumePerBlock),
	}))
	native, err := sim.CalcAmtOut(poolPkg.CalcAmtOutParams{
		TokenAmountIn: poolPkg.TokenAmt{Token: parityTokenIn, Amount: *uint256.MustFromBig(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))},
		TokenOut:      parityTokenOut,
		Limit:         inventory,
	})
	require.NoError(t, err)
	require.Equal(t, "69741277135521199", native.TokenAmountOut.Amount.Dec())
	require.Equal(t, "244951801278800", native.Fee.Amount.Dec())
	require.Equal(t, int64(600000), native.Gas)
}
