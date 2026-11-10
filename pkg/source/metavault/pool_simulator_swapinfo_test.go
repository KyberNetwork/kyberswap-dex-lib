package metavault

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func updateBalanceOf(p *PoolSimulator, in string, amount *big.Int, res *pool.CalcAmountOutResult, swapInfo any) {
	p.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: in, Amount: amount},
		TokenAmountOut: *res.TokenAmountOut,
		Fee:            *res.Fee,
		SwapInfo:       swapInfo,
	})
}

func TestPoolSimulator_SwapInfo(t *testing.T) {
	t.Parallel()

	// a vault that was donated to (credited != sent) and a healthy one, with and without the real balances
	realA0 := new(big.Int).Add(bigStr(poolA0), big.NewInt(500_000_000_000_000)).String()
	for name, newSim := range map[string]func(t *testing.T) *PoolSimulator{
		"healthy": func(t *testing.T) *PoolSimulator {
			return newRealBalanceTestSimulator(t, balancesHealthy, balancesHealthy)
		},
		"donation": func(t *testing.T) *PoolSimulator {
			return newRealBalanceTestSimulator(t, bal(realA0, poolA2, poolA10), balancesHealthy)
		},
		"real balances unknown": func(t *testing.T) *PoolSimulator { return newRealBalanceTestSimulator(t, ``, ``) },
	} {
		t.Run(name, func(t *testing.T) {
			viaInfo, viaQuote := newSim(t), newSim(t)
			for _, hop := range [][2]string{{"A0", "A2"}, {"A2", "A10"}, {"A0", "A10"}, {"A2", "A0"}} {
				amount := big.NewInt(sent)
				res, err := calcAmountOutOf(viaInfo, hop[0], hop[1], amount)
				require.NoError(t, err)
				info, ok := res.SwapInfo.(SwapInfo)
				require.True(t, ok, "SwapInfo has the type %T", res.SwapInfo)

				// the SwapInfo holds what the quote computed
				credited, err := viaInfo.vault.CreditedAmountIn(hop[0], amount)
				require.NoError(t, err)
				assert.Equal(t, credited, info.CreditedAmountIn.ToBig())
				assert.Positive(t, info.UsdmAmount.Sign())

				// consuming it ends where computing the amounts again ends
				updateBalanceOf(viaInfo, hop[0], amount, res, res.SwapInfo)
				resQuote, err := calcAmountOutOf(viaQuote, hop[0], hop[1], amount)
				require.NoError(t, err)
				assert.Equal(t, resQuote.TokenAmountOut.Amount, res.TokenAmountOut.Amount)
				updateBalanceOf(viaQuote, hop[0], amount, resQuote, nil)
				assert.Equal(t, viaQuote.vault, viaInfo.vault, hop)
			}
		})
	}

	t.Run("UpdateBalance applies the SwapInfo, it does not compute the amounts again", func(t *testing.T) {
		p := newRealBalanceTestSimulator(t, balancesHealthy, balancesHealthy)
		amount := big.NewInt(sent)
		res, err := calcAmountOutOf(p, "A0", "A2", amount)
		require.NoError(t, err)
		credited, usdm := uint256.NewInt(7), uint256.NewInt(9)
		poolA0Before, usdmA0Before := p.vault.PoolAmounts["A0"], p.vault.USDMAmounts["A0"]
		updateBalanceOf(p, "A0", amount, res, SwapInfo{CreditedAmountIn: *credited, UsdmAmount: *usdm})
		assert.Equal(t, new(big.Int).Add(poolA0Before, big.NewInt(7)), p.vault.PoolAmounts["A0"])
		assert.Equal(t, new(big.Int).Add(usdmA0Before, big.NewInt(9)), p.vault.USDMAmounts["A0"])
	})

	t.Run("a missing or foreign SwapInfo falls back to computing the amounts", func(t *testing.T) {
		ref := newRealBalanceTestSimulator(t, balancesHealthy, balancesHealthy)
		res := swapAndUpdate(t, ref, "A0", "A2", big.NewInt(sent))

		for name, swapInfo := range map[string]any{
			"nil":               nil,
			"pointer":           &SwapInfo{CreditedAmountIn: *uint256.NewInt(7)},
			"nil pointer":       (*SwapInfo)(nil),
			"other type":        struct{ C string }{"7"},
			"decoded from JSON": map[string]any{"c": "7", "u": "9"},
		} {
			p := newRealBalanceTestSimulator(t, balancesHealthy, balancesHealthy)
			assert.NotPanics(t, func() { updateBalanceOf(p, "A0", big.NewInt(sent), res, swapInfo) }, name)
			assert.Equal(t, ref.vault, p.vault, name)
		}
	})

	t.Run("the state does not share memory with the SwapInfo", func(t *testing.T) {
		p := newRealBalanceTestSimulator(t, balancesHealthy, balancesHealthy)
		amount := big.NewInt(sent)
		res, err := calcAmountOutOf(p, "A0", "A2", amount)
		require.NoError(t, err)
		info := res.SwapInfo.(SwapInfo)
		before := info
		updateBalanceOf(p, "A0", amount, res, info)
		p.vault.PoolAmounts["A0"].Add(p.vault.PoolAmounts["A0"], big.NewInt(1))
		p.vault.USDMAmounts["A0"].Add(p.vault.USDMAmounts["A0"], big.NewInt(1))
		assert.Equal(t, before, res.SwapInfo)
	})

	t.Run("a failed quote has no SwapInfo", func(t *testing.T) {
		p := newRealBalanceTestSimulator(t, balancesA0A2Zero, balancesA0A2Zero)
		res, err := calcAmountOut(p, "A0", "A2")
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}

func TestSwapInfo_JSON(t *testing.T) {
	t.Parallel()

	// the amounts are internal state of the vault: they are not part of the swap extra in the route response
	p := newRealBalanceTestSimulator(t, balancesHealthy, balancesHealthy)
	res, err := calcAmountOut(p, "A0", "A2")
	require.NoError(t, err)
	require.IsType(t, SwapInfo{}, res.SwapInfo)
	b, err := json.Marshal(res.SwapInfo)
	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(b))

	b, err = json.Marshal(struct {
		Extra any `json:"extra"`
	}{res.SwapInfo})
	require.NoError(t, err)
	assert.JSONEq(t, `{"extra":{}}`, string(b))
}

func TestNewSwapInfo(t *testing.T) {
	t.Parallel()

	tooBig := new(big.Int).Lsh(big.NewInt(1), 256)
	assert.Nil(t, newSwapInfo(tooBig, big.NewInt(1)))
	assert.Nil(t, newSwapInfo(big.NewInt(1), tooBig))
	assert.Equal(t, SwapInfo{CreditedAmountIn: *uint256.NewInt(1), UsdmAmount: *uint256.NewInt(2)},
		newSwapInfo(big.NewInt(1), big.NewInt(2)))
}

// The swap fee reads the vault through vaultUtils: after CloneState it must read the cloned vault, so that an
// UpdateBalance on the clone moves the fee of the clone and leaves the original as it was.
func TestPoolSimulator_CloneState_VaultUtils(t *testing.T) {
	t.Parallel()

	feeBps := func(p *PoolSimulator) string {
		return p.vaultUtils.GetSwapFeeBasisPoints("A0", "A2", bigStr("100000000000000000000000")).String()
	}

	orig := newRealBalanceTestSimulator(t, balancesHealthy, balancesHealthy)
	origFee := feeBps(orig)
	origVault := newRealBalanceTestSimulator(t, balancesHealthy, balancesHealthy).vault

	cloned := orig.CloneState().(*PoolSimulator)
	assert.Same(t, cloned.vault, cloned.vaultUtils.vault)
	assert.Same(t, orig.vault, orig.vaultUtils.vault)
	assert.NotSame(t, orig.vaultUtils, cloned.vaultUtils)

	// move a lot of USDM into A0 (and out of A2) on the clone only (the credited pool amount stays small, so quotes work)
	res, err := calcAmountOutOf(cloned, "A0", "A2", big.NewInt(sent))
	require.NoError(t, err)
	big1e23 := uint256.MustFromDecimal("100000000000000000000000")
	updateBalanceOf(cloned, "A0", big.NewInt(sent), res, SwapInfo{CreditedAmountIn: *uint256.NewInt(sent), UsdmAmount: *big1e23})

	// the clone quotes with its own state: the same as a simulator that went through the same update
	ref := newRealBalanceTestSimulator(t, balancesHealthy, balancesHealthy)
	updateBalanceOf(ref, "A0", big.NewInt(sent), res, SwapInfo{CreditedAmountIn: *uint256.NewInt(sent), UsdmAmount: *big1e23})
	require.NotEqual(t, origFee, feeBps(ref), "the update must change the fee, or this test proves nothing")
	assert.Equal(t, feeBps(ref), feeBps(cloned))
	resCloned, err := calcAmountOutOf(cloned, "A0", "A2", big.NewInt(sent))
	require.NoError(t, err)
	resRef, err := calcAmountOutOf(ref, "A0", "A2", big.NewInt(sent))
	require.NoError(t, err)
	assert.Equal(t, resRef.TokenAmountOut.Amount, resCloned.TokenAmountOut.Amount)
	assert.Equal(t, resRef.Fee.Amount, resCloned.Fee.Amount)

	// the original did not move
	assert.Equal(t, origFee, feeBps(orig))
	assert.Equal(t, origVault, orig.vault)
}
