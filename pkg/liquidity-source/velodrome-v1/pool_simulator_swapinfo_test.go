package velodromev1

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// updateBalanceWithoutSwapInfo is what a caller that lost the SwapInfo does.
func updateBalanceWithoutSwapInfo(sim *PoolSimulator, in int, amountIn *big.Int, res *poolpkg.CalcAmountOutResult) {
	sim.UpdateBalance(poolpkg.UpdateBalanceParams{
		TokenAmountIn:  poolpkg.TokenAmount{Token: tokenOf(in), Amount: amountIn},
		TokenAmountOut: *res.TokenAmountOut,
		Fee:            *res.Fee,
	})
}

func simState(sim *PoolSimulator) [4]*big.Int {
	state := [4]*big.Int{}
	for i := range 2 {
		state[i] = sim.reserves[i].ToBig()
		if sim.balances != nil {
			state[2+i] = sim.balances[i].ToBig()
		}
	}
	return state
}

func TestPoolSimulator_SwapInfo(t *testing.T) {
	t.Parallel()

	for _, tp := range balanceTestPools {
		t.Run(tp.name, func(t *testing.T) {
			reserves := []*big.Int{bi(tp.reserves[0]), bi(tp.reserves[1])}
			donated := []*big.Int{new(big.Int).Add(reserves[0], big.NewInt(3_000_000)), reserves[1]}

			for name, balances := range map[string][]*big.Int{"balances == reserves": reserves, "donation": donated} {
				t.Run(name, func(t *testing.T) {
					viaInfo := newBalanceTestSimulator(t, tp, reserves, balances)
					viaSwap := newBalanceTestSimulator(t, tp, reserves, balances)

					for _, hop := range []struct {
						in     int
						amount *big.Int
					}{{0, new(big.Int).Div(reserves[0], big.NewInt(50))}, {1, new(big.Int).Div(reserves[1], big.NewInt(40))},
						{0, new(big.Int).Div(reserves[0], big.NewInt(90))}} {
						res, err := quote(viaInfo, hop.in, hop.amount)
						require.NoError(t, err)
						info, ok := res.SwapInfo.(SwapInfo)
						require.True(t, ok, "SwapInfo has the type %T", res.SwapInfo)
						// the same quote as without the SwapInfo
						resSwap, err := quote(viaSwap, hop.in, hop.amount)
						require.NoError(t, err)
						assert.Equal(t, resSwap.TokenAmountOut.Amount, res.TokenAmountOut.Amount)
						assert.Equal(t, resSwap.Fee.Amount, res.Fee.Amount)

						updateBalance(viaInfo, hop.in, hop.amount, res)
						updateBalanceWithoutSwapInfo(viaSwap, hop.in, hop.amount, resSwap)
						// consuming the SwapInfo ends where running the swap again ends
						assert.Equal(t, simState(viaSwap), simState(viaInfo))
						assert.Equal(t, info.NewBalance0.ToBig(), viaInfo.balances[0].ToBig())
						assert.Equal(t, info.NewBalance1.ToBig(), viaInfo.balances[1].ToBig())
					}
				})
			}
		})
	}

	tp := balanceTestPools[0]
	reserves := []*big.Int{bi(tp.reserves[0]), bi(tp.reserves[1])}
	amountIn := big.NewInt(1_000_000_000)

	t.Run("UpdateBalance applies the SwapInfo, it does not run the swap again", func(t *testing.T) {
		sim := newBalanceTestSimulator(t, tp, reserves, reserves)
		res, err := quote(sim, 0, amountIn)
		require.NoError(t, err)
		res.SwapInfo = SwapInfo{NewBalance0: *uint256.NewInt(11), NewBalance1: *uint256.NewInt(22)}
		updateBalance(sim, 0, amountIn, res)
		assert.Equal(t, [4]*big.Int{big.NewInt(11), big.NewInt(22), big.NewInt(11), big.NewInt(22)}, simState(sim))
	})

	t.Run("the state does not share memory with the SwapInfo", func(t *testing.T) {
		sim := newBalanceTestSimulator(t, tp, reserves, reserves)
		res, err := quote(sim, 0, amountIn)
		require.NoError(t, err)
		info := res.SwapInfo.(SwapInfo)
		before := info
		updateBalance(sim, 0, amountIn, res)
		sim.reserves[0].AddUint64(sim.reserves[0], 1)
		sim.balances[1].AddUint64(sim.balances[1], 1)
		assert.Equal(t, before, res.SwapInfo)
		assert.NotSame(t, sim.reserves[0], sim.balances[0])
		// UpdateBalance on a clone leaves the original and the SwapInfo as they were
		orig := newBalanceTestSimulator(t, tp, reserves, reserves)
		cloned := orig.CloneState().(*PoolSimulator)
		updateBalance(cloned, 0, amountIn, res)
		assert.Equal(t, [4]*big.Int{reserves[0], reserves[1], reserves[0], reserves[1]}, simState(orig))
	})

	t.Run("a missing or foreign SwapInfo falls back to running the swap", func(t *testing.T) {
		ref := newBalanceTestSimulator(t, tp, reserves, reserves)
		res, err := quote(ref, 0, amountIn)
		require.NoError(t, err)
		updateBalanceWithoutSwapInfo(ref, 0, amountIn, res)

		for name, swapInfo := range map[string]any{
			"nil":               nil,
			"pointer":           &SwapInfo{NewBalance0: *uint256.NewInt(1)},
			"nil pointer":       (*SwapInfo)(nil),
			"other type":        struct{ B0 string }{"1"},
			"decoded from JSON": map[string]any{"b0": "1", "b1": "2"},
		} {
			sim := newBalanceTestSimulator(t, tp, reserves, reserves)
			resSim, err := quote(sim, 0, amountIn)
			require.NoError(t, err, name)
			resSim.SwapInfo = swapInfo
			assert.NotPanics(t, func() { updateBalance(sim, 0, amountIn, resSim) }, name)
			assert.Equal(t, simState(ref), simState(sim), name)
		}
	})

	t.Run("CalcAmountIn carries the SwapInfo too", func(t *testing.T) {
		viaInfo := newBalanceTestSimulator(t, tp, reserves, reserves)
		viaSwap := newBalanceTestSimulator(t, tp, reserves, reserves)
		amountOut := big.NewInt(500_000_000)
		params := poolpkg.CalcAmountInParams{
			TokenAmountOut: poolpkg.TokenAmount{Token: tokenB, Amount: amountOut},
			TokenIn:        tokenA,
		}
		res, err := viaInfo.CalcAmountIn(params)
		require.NoError(t, err)
		_, ok := res.SwapInfo.(SwapInfo)
		require.True(t, ok)
		updateParams := poolpkg.UpdateBalanceParams{
			TokenAmountIn:  *res.TokenAmountIn,
			TokenAmountOut: params.TokenAmountOut,
			Fee:            *res.Fee,
		}
		viaSwap.UpdateBalance(updateParams)
		updateParams.SwapInfo = res.SwapInfo
		viaInfo.UpdateBalance(updateParams)
		assert.Equal(t, simState(viaSwap), simState(viaInfo))
	})

	t.Run("unknown balances: no SwapInfo, and a SwapInfo that comes anyway is ignored", func(t *testing.T) {
		legacy := newBalanceTestSimulator(t, tp, reserves, nil)
		res, err := quote(legacy, 0, amountIn)
		require.NoError(t, err)
		assert.Nil(t, res.SwapInfo)
		resIn, err := legacy.CalcAmountIn(poolpkg.CalcAmountInParams{
			TokenAmountOut: poolpkg.TokenAmount{Token: tokenB, Amount: big.NewInt(500_000_000)}, TokenIn: tokenA})
		require.NoError(t, err)
		assert.Nil(t, resIn.SwapInfo)

		ref := newBalanceTestSimulator(t, tp, reserves, nil)
		updateBalanceWithoutSwapInfo(ref, 0, amountIn, res)
		res.SwapInfo = SwapInfo{NewBalance0: *uint256.NewInt(11), NewBalance1: *uint256.NewInt(22)}
		updateBalance(legacy, 0, amountIn, res)
		assert.Equal(t, simState(ref), simState(legacy))
		assert.Nil(t, legacy.balances)
	})
}

func TestSwapInfo_JSON(t *testing.T) {
	t.Parallel()

	// the balances are internal state of the pair: they are not part of the swap extra in the route response
	info := SwapInfo{NewBalance0: *uint256.MustFromDecimal("1000000000000"), NewBalance1: *uint256.MustFromDecimal("2999999999000")}
	b, err := json.Marshal(info)
	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(b))

	b, err = json.Marshal(struct {
		Extra any `json:"extra"`
	}{info})
	require.NoError(t, err)
	assert.JSONEq(t, `{"extra":{}}`, string(b))
}
