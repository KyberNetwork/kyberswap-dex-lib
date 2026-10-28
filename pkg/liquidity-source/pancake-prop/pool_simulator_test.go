package pancakeprop

import (
	"math/big"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

const (
	usdt  = "0x55d398326f99059ff775485246999027b3197955"
	wbnb  = "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"
	vault = "0x00fc010f0d2e5010635a0c8702f8b9779fa6befe"
	e18   = "1000000000000000000"
)

// newSim is BSC pair 1 (USDT/WBNB) at block 128674036 with the MM's live price word and the
// vault balances read at that block.
func newSim(t testing.TB, mutate func(*Extra), staleCheck bool) *PoolSimulator {
	t.Helper()
	extra := Extra{
		Packed:      *uint256.MustFromHex(livePair1),
		MaxInput:    [2]uint256.Int{*uint256.MustFromDecimal("5000000000000000000000"), *uint256.MustFromDecimal("7500000000000000000")},
		ExpiredAtMs: uint64(time.Now().Add(time.Hour).UnixMilli()),
	}
	if mutate != nil {
		mutate(&extra)
	}
	extraJSON, err := json.Marshal(&extra)
	require.NoError(t, err)
	staticJSON, err := json.Marshal(StaticExtra{Entry: "0x12b76204f5a1b390002a73e3d869196f0e6e8b1c", Vault: vault, PairID: 1})
	require.NoError(t, err)
	sim, err := NewPoolSimulator(pool.FactoryParams{
		EntityPool: entity.Pool{
			Address: "0xabc", Exchange: DexType, Type: DexType, BlockNumber: 128674036,
			Tokens:      []*entity.PoolToken{{Address: usdt}, {Address: wbnb}},
			Reserves:    entity.PoolReserves{"24244540955976161967236", "4130696923886726262"},
			Extra:       string(extraJSON),
			StaticExtra: string(staticJSON),
		},
		Opts: pool.FactoryOpts{StaleCheck: staleCheck},
	})
	require.NoError(t, err)
	return sim
}

func calc(sim *PoolSimulator, tokenIn, tokenOut, amount string) (*pool.CalcAmountOutResult, error) {
	a, _ := new(big.Int).SetString(amount, 10)
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: a}, TokenOut: tokenOut,
	})
}

func usdtIn(n int64) string { return new(big.Int).Mul(big.NewInt(n), decBig(e18)).String() }

func decBig(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic(s)
	}
	return v
}

// Quotes of the deployed contract at the same block (eth_call on a fork).
func TestCalcAmountOutMatchesChain(t *testing.T) {
	sim := newSim(t, nil, false)
	for usdtAmt, want := range map[int64]string{
		1:    "1369727795562197",
		100:  "136972779556219708",
		1000: "1369727795562197084",
		1500: "2054591693343295627",
		2000: "2739385177787380682",
		2500: "3423756182209384871",
		3000: "4107282226587227294",
	} {
		res, err := calc(sim, usdt, wbnb, usdtIn(usdtAmt))
		require.NoError(t, err, usdtAmt)
		assert.Equal(t, want, res.TokenAmountOut.Amount.String(), "%d USDT", usdtAmt)
	}
	res, err := calc(sim, wbnb, usdt, e18)
	require.NoError(t, err)
	assert.Equal(t, "728759116794884613659", res.TokenAmountOut.Amount.String())
}

func TestCalcAmountOutLimits(t *testing.T) {
	sim := newSim(t, nil, false)
	// 3500 USDT is within the epoch capacity but needs more WBNB than the vault holds.
	_, err := calc(sim, usdt, wbnb, usdtIn(3500))
	assert.ErrorIs(t, err, pool.ErrNotEnoughInventory)
	_, err = calc(sim, usdt, wbnb, usdtIn(5000))
	assert.ErrorIs(t, err, pool.ErrNotEnoughInventory, "5000 is exactly the capacity ceiling, so only the vault limits it")
	_, err = calc(sim, usdt, wbnb, usdtIn(5001))
	assert.ErrorIs(t, err, ErrExceedsCapacity)

	_, err = calc(sim, usdt, wbnb, "0")
	assert.ErrorIs(t, err, ErrZeroAmount)
	_, err = calc(sim, usdt, usdt, "1")
	assert.ErrorIs(t, err, ErrInvalidToken)
	_, err = calc(sim, usdt, wbnb, "1")
	assert.ErrorIs(t, err, ErrZeroOutput, "1 wei of USDT rounds to no WBNB")
}

func TestPausedAndExpiry(t *testing.T) {
	_, err := calc(newSim(t, func(e *Extra) { e.Paused = true }, false), usdt, wbnb, usdtIn(1))
	assert.ErrorIs(t, err, ErrPaused)

	past := func(e *Extra) { e.ExpiredAtMs = uint64(time.Now().Add(-time.Second).UnixMilli()) }
	_, err = calc(newSim(t, past, true), usdt, wbnb, usdtIn(1))
	assert.ErrorIs(t, err, ErrStalePrice)
	// The liquidity indexer builds simulators without the staleness check.
	_, err = calc(newSim(t, past, false), usdt, wbnb, usdtIn(1))
	assert.NoError(t, err)
}

// Consuming input moves the pair along its depth curve: later input gets the worse price, but
// pricing integrates the curve, so splitting an order gives the same total as one order.
func TestUpdateBalanceAdvancesEpochCumulative(t *testing.T) {
	sim := newSim(t, nil, false)
	whole, err := calc(sim, usdt, wbnb, usdtIn(3000))
	require.NoError(t, err)

	first, err := calc(sim, usdt, wbnb, usdtIn(1500))
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn: pool.TokenAmount{Token: usdt, Amount: decBig(usdtIn(1500))}, TokenAmountOut: *first.TokenAmountOut,
		SwapInfo: first.SwapInfo,
	})
	assert.Equal(t, usdtIn(1500), sim.Extra.Cum[0].Dec())
	assert.Equal(t, "0", sim.Extra.Cum[1].Dec(), "the other direction keeps its own cumulative")

	second, err := calc(sim, usdt, wbnb, usdtIn(1500))
	require.NoError(t, err)
	assert.Equal(t, whole.TokenAmountOut.Amount.String(),
		new(big.Int).Add(first.TokenAmountOut.Amount, second.TokenAmountOut.Amount).String())
	assert.Equal(t, -1, second.TokenAmountOut.Amount.Cmp(first.TokenAmountOut.Amount), "second half is on the curve")

	// 1500 + 3500 would pass the 5000 ceiling.
	_, err = calc(sim, usdt, wbnb, usdtIn(3501))
	assert.ErrorIs(t, err, ErrExceedsCapacity)
}

func TestSharedVaultInventory(t *testing.T) {
	sim := newSim(t, nil, false)
	limits := sim.CalculateLimit()
	assert.Equal(t, "4130696923886726262", limits[vault+":"+wbnb].String())
	inv := swaplimit.NewInventory(DexType, limits)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: usdt, Amount: decBig(usdtIn(3000))}, TokenOut: wbnb, Limit: inv,
	})
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn: pool.TokenAmount{Token: usdt, Amount: decBig(usdtIn(3000))}, TokenAmountOut: *res.TokenAmountOut,
		SwapInfo: res.SwapInfo, SwapLimit: inv,
	})
	// Another pair of the same vault sees the reduced WBNB inventory.
	assert.Equal(t, new(big.Int).Sub(decBig("4130696923886726262"), res.TokenAmountOut.Amount).String(),
		inv.GetLimit(vault+":"+wbnb).String())

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: usdt, Amount: decBig(usdtIn(1000))}, TokenOut: wbnb, Limit: inv,
	})
	assert.ErrorIs(t, err, pool.ErrNotEnoughInventory)
}

func TestCloneStateIsIndependent(t *testing.T) {
	sim := newSim(t, nil, false)
	clone := sim.CloneState().(*PoolSimulator)
	res, err := calc(clone, usdt, wbnb, usdtIn(2000))
	require.NoError(t, err)
	clone.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})

	assert.True(t, sim.Extra.Cum[0].IsZero())
	assert.Equal(t, "4130696923886726262", sim.Info.Reserves[1].String())
	assert.NotEqual(t, sim.Info.Reserves[1].String(), clone.Info.Reserves[1].String())
}

func TestMetaAndSlot(t *testing.T) {
	sim := newSim(t, nil, false)
	assert.Equal(t, PoolMeta{BlockNumber: 128674036, Entry: "0x12b76204f5a1b390002a73e3d869196f0e6e8b1c"}, sim.GetMetaInfo(usdt, wbnb))
	// Mapping slot of pair 1's price word, as traced on the deployed pricing module.
	assert.Equal(t, "0xabd6e7cb50984ff9c2f3e18a2660c3353dadf4e3291deeb275dae2cd1e44fe05", priceSlot(1).Hex())
}
