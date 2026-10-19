package erc4626

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
)

// depositPool builds a 1:1 deposit-only vault; vault token is index 0, asset index 1.
func depositPool(t *testing.T, mod func(*Extra)) *PoolSimulator {
	t.Helper()
	dr := make([]*uint256.Int, len(PrefetchAmounts))
	for i, a := range PrefetchAmounts {
		dr[i] = a.Clone()
	}
	extra := Extra{Gas: Gas{Deposit: 80000}, DepositRates: dr}
	mod(&extra)
	b, err := json.Marshal(extra)
	require.NoError(t, err)
	sim, err := NewPoolSimulator(entity.Pool{
		Address: "0xvault", Exchange: DexType, Type: DexType, Reserves: entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{{Address: "0xvault", Swappable: true}, {Address: "0xasset", Swappable: true}},
		Extra:  string(b),
	})
	require.NoError(t, err)
	return sim
}

func deposit(sim *PoolSimulator, amt int64) (*pool.CalcAmountOutResult, error) {
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xasset", Amount: big.NewInt(amt)},
		TokenOut:      "0xvault",
	})
}

func TestDepositsPaused(t *testing.T) {
	t.Parallel()
	sim := depositPool(t, func(e *Extra) { e.DepositsPaused = true })
	_, err := deposit(sim, 1e9)
	assert.ErrorIs(t, err, ErrERC4626DepositsPaused)
	_, err = sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: "0xvault", Amount: big.NewInt(1e9)}, TokenIn: "0xasset",
	})
	assert.ErrorIs(t, err, ErrERC4626DepositsPaused)

	_, err = deposit(depositPool(t, func(*Extra) {}), 1e9)
	assert.NoError(t, err)
}

func TestMintHeadroom(t *testing.T) {
	t.Parallel()
	// maxDeposit is far above the headroom, as upAVAX: maxDeposit 1e26 but only ~30k WAVAX of supply room.
	sim := depositPool(t, func(e *Extra) {
		e.MaxDeposit = uint256.NewInt(1e15)
		e.MintHeadroom = uint256.NewInt(1e9)
	})
	_, err := deposit(sim, 1e9+1)
	assert.ErrorIs(t, err, ErrERC4626MintMoreThanMax)
	_, err = sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: "0xvault", Amount: big.NewInt(1e9 + 1)}, TokenIn: "0xasset",
	})
	assert.ErrorIs(t, err, ErrERC4626MintMoreThanMax)

	res, err := deposit(sim, 1e9)
	require.NoError(t, err)
	assert.Equal(t, int64(1e9), res.TokenAmountOut.Amount.Int64())

	// zero headroom (cap reached) must reject, not read as "no cap"
	b, _ := json.Marshal(Extra{MintHeadroom: uint256.NewInt(0)})
	var rt Extra
	require.NoError(t, json.Unmarshal(b, &rt))
	require.NotNil(t, rt.MintHeadroom)
	full := depositPool(t, func(e *Extra) { e.MintHeadroom = uint256.NewInt(0) })
	_, err = deposit(full, 1)
	assert.ErrorIs(t, err, ErrERC4626MintMoreThanMax)
}

func TestUpdateBalanceSplitRoute(t *testing.T) {
	t.Parallel()
	sim := depositPool(t, func(e *Extra) {
		e.MaxDeposit = uint256.NewInt(1000)
		e.MintHeadroom = uint256.NewInt(600)
	})
	clone := sim.CloneState().(*PoolSimulator)

	// 2 x 400 fits maxDeposit (1000) but the second exceeds the 600 mint headroom.
	res, err := deposit(clone, 400)
	require.NoError(t, err)
	clone.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xasset", Amount: big.NewInt(400)}, TokenAmountOut: *res.TokenAmountOut,
		Fee: *res.Fee, SwapInfo: res.SwapInfo,
	})
	assert.Equal(t, uint64(200), clone.MintHeadroom.Uint64())
	assert.Equal(t, uint64(600), clone.MaxDeposit.Uint64())
	_, err = deposit(clone, 400)
	assert.ErrorIs(t, err, ErrERC4626MintMoreThanMax)

	// clone must not leak into the original
	assert.Equal(t, uint64(600), sim.MintHeadroom.Uint64())
	assert.Equal(t, uint64(1000), sim.MaxDeposit.Uint64())
}

func TestUpdateEntityStateKeepsLimits(t *testing.T) {
	t.Parallel()
	var p entity.Pool
	require.NoError(t, UpdateEntityState(&p, VaultCfg{Gas: GasCfg{Deposit: 1}}, &PoolState{
		MaxDeposit: big.NewInt(5), MaxRedeem: big.NewInt(0), MinDeposit: big.NewInt(0), TotalAssets: big.NewInt(7),
		Paused: true, MintHeadroom: big.NewInt(0),
	}))
	var e Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
	assert.True(t, e.DepositsPaused)
	require.NotNil(t, e.MintHeadroom)
	assert.True(t, e.MintHeadroom.IsZero())
}

// UpdateBalance swaps in a new pointer, so it cannot show shared backing state. Mutating the clone's
// headroom in place does: it reaches the original unless CloneState copied it.
func TestCloneStateCopiesMintHeadroom(t *testing.T) {
	t.Parallel()
	sim := depositPool(t, func(e *Extra) { e.MintHeadroom = uint256.NewInt(600) })
	clone := sim.CloneState().(*PoolSimulator)

	assert.NotSame(t, sim.MintHeadroom, clone.MintHeadroom)
	clone.MintHeadroom.SetUint64(1)
	assert.Equal(t, uint64(600), sim.MintHeadroom.Uint64())
	_, err := deposit(sim, 600)
	assert.NoError(t, err)

	// untracked headroom stays untracked in the clone
	assert.Nil(t, depositPool(t, func(*Extra) {}).CloneState().(*PoolSimulator).MintHeadroom)
}

// A failed pause/cap read saves nothing, so the pool ages; MaxAge must then stop deposit quotes.
func TestMaxAgeStale(t *testing.T) {
	t.Parallel()
	now := time.Now().Unix()
	for _, tc := range []struct {
		name        string
		maxAge, age int64
		wantStale   bool
	}{
		{"fresh", 60, 10, false},
		{"stale", 60, 120, true},
		{"off when zero", 0, 100000, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sim := depositPool(t, func(e *Extra) { e.MaxAge = tc.maxAge })
			sim.poolTimestamp = now - tc.age

			_, err := deposit(sim, 1e9)
			inParams := pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{Token: "0xvault", Amount: big.NewInt(1e9)}, TokenIn: "0xasset",
			}
			_, errIn := sim.CalcAmountIn(inParams)
			if tc.wantStale {
				assert.ErrorIs(t, err, ErrERC4626StalePoolData)
				assert.ErrorIs(t, errIn, ErrERC4626StalePoolData)
			} else {
				assert.NoError(t, err)
				assert.NoError(t, errIn)
			}
			// CloneState must carry both fields.
			clone := sim.CloneState().(*PoolSimulator)
			assert.Equal(t, sim.MaxAge, clone.MaxAge)
			assert.Equal(t, sim.poolTimestamp, clone.poolTimestamp)
		})
	}
}

// The yaml maxAge must reach Extra, or the simulator check never arms.
func TestUpdateEntityStateWritesMaxAge(t *testing.T) {
	t.Parallel()
	z := big.NewInt(0)
	state := &PoolState{MaxDeposit: z, MaxRedeem: z, MinDeposit: z, TotalAssets: z, TotalSupply: z}
	p := entity.Pool{}
	require.NoError(t, UpdateEntityState(&p, VaultCfg{MaxAge: 60}, state))
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	assert.EqualValues(t, 60, extra.MaxAge)
}

// The indexer builds simulators with Opts.StaleCheck=false and must still quote deposits from an aged pool;
// route finding (true) must reject them.
func TestFactoryStaleCheck(t *testing.T) {
	t.Parallel()
	for _, staleCheck := range []bool{false, true} {
		old := depositPool(t, func(e *Extra) { e.MaxAge = 60 })
		ep := entity.Pool{
			Address: "0xvault", Exchange: DexType, Type: DexType, Reserves: entity.PoolReserves{"0", "0"},
			Tokens:    []*entity.PoolToken{{Address: "0xvault", Swappable: true}, {Address: "0xasset", Swappable: true}},
			Timestamp: time.Now().Unix() - 3600,
		}
		b, err := json.Marshal(old.Extra)
		require.NoError(t, err)
		ep.Extra = string(b)

		iface, err := pool.Factory(DexType)(pool.FactoryParams{EntityPool: ep, Opts: pool.FactoryOpts{StaleCheck: staleCheck}})
		require.NoError(t, err)
		sim := iface.CloneState().(*PoolSimulator)
		_, err = deposit(sim, 1e9)
		if staleCheck {
			assert.ErrorIs(t, err, ErrERC4626StalePoolData)
		} else {
			assert.NoError(t, err)
		}
	}
}
