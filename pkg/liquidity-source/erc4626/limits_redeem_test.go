package erc4626

import (
	"math/big"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// bothPool builds a vault with both directions; shares = assets / div on deposit, assets = shares * div on
// redeem. Vault token is index 0, asset index 1.
func bothPool(t *testing.T, div int64, mod func(*Extra)) *PoolSimulator {
	t.Helper()
	dr := make([]*uint256.Int, len(PrefetchAmounts))
	rr := make([]*uint256.Int, len(PrefetchAmounts))
	for i, a := range PrefetchAmounts {
		dr[i] = new(uint256.Int).Div(a, uint256.NewInt(uint64(div)))
		rr[i] = new(uint256.Int).Mul(a, uint256.NewInt(uint64(div)))
	}
	extra := Extra{Gas: Gas{Deposit: 80000, Redeem: 70000}, DepositRates: dr, RedeemRates: rr}
	mod(&extra)
	b, err := json.Marshal(extra)
	require.NoError(t, err)
	sim, err := NewPoolSimulator(entity.Pool{
		Address: "0xvault", Exchange: DexType, Type: DexType, Reserves: entity.PoolReserves{"0", "0"},
		Tokens:    []*entity.PoolToken{{Address: "0xvault", Swappable: true}, {Address: "0xasset", Swappable: true}},
		Extra:     string(b),
		Timestamp: time.Now().Unix(),
	})
	require.NoError(t, err)
	return sim
}

func redeem(sim *PoolSimulator, amt int64) (*pool.CalcAmountOutResult, error) {
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xvault", Amount: big.NewInt(amt)},
		TokenOut:      "0xasset",
	})
}

func redeemIn(sim *PoolSimulator, out int64) (*pool.CalcAmountInResult, error) {
	return sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: "0xasset", Amount: big.NewInt(out)}, TokenIn: "0xvault",
	})
}

func TestResolveLimitsRedeemsPaused(t *testing.T) {
	t.Parallel()
	// redeem-only vault (no deposit gas): the redeem pause is still resolved
	cfg := VaultCfg{Gas: GasCfg{Redeem: 1}, RedeemsPausedMethod: "paused"}
	s := PoolState{RawRedeemsPaused: big.NewInt(1)}
	require.NoError(t, s.ResolveLimits(cfg))
	assert.True(t, s.RedeemsPaused)

	s = PoolState{RawRedeemsPaused: big.NewInt(0)}
	require.NoError(t, s.ResolveLimits(cfg))
	assert.False(t, s.RedeemsPaused)

	// a reverted read must fail the refresh, not read as "not paused"
	s = PoolState{}
	assert.ErrorIs(t, s.ResolveLimits(cfg), ErrReadFailed)

	// deposit-enabled vault: both pauses resolve independently
	both := VaultCfg{Gas: GasCfg{Deposit: 1, Redeem: 1}, DepositsPausedMethod: "paused", RedeemsPausedMethod: "paused"}
	s = PoolState{RawPaused: big.NewInt(0), RawRedeemsPaused: big.NewInt(1)}
	require.NoError(t, s.ResolveLimits(both))
	assert.False(t, s.Paused)
	assert.True(t, s.RedeemsPaused)
	s = PoolState{RawPaused: big.NewInt(0)}
	assert.ErrorIs(t, s.ResolveLimits(both), ErrReadFailed)

	// not configured, or no redeem support: nothing read, nothing tracked
	s = PoolState{}
	require.NoError(t, s.ResolveLimits(VaultCfg{Gas: GasCfg{Deposit: 1, Redeem: 1}}))
	assert.False(t, s.RedeemsPaused)
	require.NoError(t, s.ResolveLimits(VaultCfg{Gas: GasCfg{Deposit: 1}, RedeemsPausedMethod: "paused"}))
}

func TestAddRedeemLimitCalls(t *testing.T) {
	t.Parallel()
	collect := func(cfg VaultCfg) (methods []string) {
		AddRedeemLimitCalls(func(c *ethrpc.Call, _ []any) { methods = append(methods, c.Method) }, "0xvault", cfg, &PoolState{})
		return
	}
	assert.Equal(t, []string{"paused"}, collect(VaultCfg{Gas: GasCfg{Redeem: 1}, RedeemsPausedMethod: "paused"}))
	assert.Empty(t, collect(VaultCfg{Gas: GasCfg{Redeem: 1}}))
	assert.Empty(t, collect(VaultCfg{Gas: GasCfg{Deposit: 1}, RedeemsPausedMethod: "paused"}))
}

func TestNormalizeState(t *testing.T) {
	t.Parallel()
	state := func(maxDeposit *big.Int) *PoolState {
		return &PoolState{MaxDeposit: maxDeposit, TotalAssets: big.NewInt(77), TotalSupply: big.NewInt(55)}
	}
	cfg := VaultCfg{Gas: GasCfg{Deposit: 1, Redeem: 1}}

	// the maxDeposit rules are ResolveMaxDeposit's: a zero stays closed, a failed read falls back to totalAssets
	s := state(big.NewInt(0))
	NormalizeState(cfg, s)
	require.NotNil(t, s.MaxDeposit)
	assert.Zero(t, s.MaxDeposit.Sign())
	s = state(nil)
	NormalizeState(cfg, s)
	assert.Equal(t, int64(77), s.MaxDeposit.Int64())

	// minDeposit 0 means no minimum
	s = state(big.NewInt(1))
	s.MinDeposit = big.NewInt(0)
	NormalizeState(cfg, s)
	assert.Nil(t, s.MinDeposit)

	// maxRedeem: 0 falls back to totalSupply, too large means no limit, redeem off clears it
	s = state(big.NewInt(1))
	s.MaxRedeem = big.NewInt(0)
	NormalizeState(cfg, s)
	assert.Equal(t, int64(55), s.MaxRedeem.Int64())
	s.MaxRedeem = new(big.Int).Add(bignum.MaxUint128, big.NewInt(1))
	NormalizeState(cfg, s)
	assert.Nil(t, s.MaxRedeem)
	s.MaxRedeem = big.NewInt(3)
	NormalizeState(VaultCfg{Gas: GasCfg{Deposit: 1}}, s)
	assert.Nil(t, s.MaxRedeem)
}

// maxDeposit 0 must travel tracker state -> Extra -> simulator and refuse every deposit, unless the vault opts out.
func TestZeroMaxDepositEndToEnd(t *testing.T) {
	t.Parallel()
	newState := func() *PoolState {
		st := &PoolState{
			MaxDeposit: big.NewInt(0), MaxRedeem: big.NewInt(0), TotalAssets: big.NewInt(1e18), TotalSupply: big.NewInt(1e18),
			DepositRates: make([]*big.Int, len(PrefetchAmounts)), RedeemRates: make([]*big.Int, len(PrefetchAmounts)),
		}
		for i, a := range PrefetchAmounts {
			st.DepositRates[i], st.RedeemRates[i] = a.ToBig(), a.ToBig()
		}
		return st
	}
	p := entity.Pool{
		Address: "0xvault", Exchange: DexType, Type: DexType,
		Tokens: []*entity.PoolToken{{Address: "0xvault", Swappable: true}, {Address: "0xasset", Swappable: true}},
	}
	sim := func(cfg VaultCfg) *PoolSimulator {
		st := newState()
		NormalizeState(cfg, st)
		require.NoError(t, UpdateEntityState(&p, cfg, st))
		s, err := NewPoolSimulator(p)
		require.NoError(t, err)
		return s
	}

	closed := sim(VaultCfg{Gas: GasCfg{Deposit: 80000, Redeem: 70000}})
	require.NotNil(t, closed.MaxDeposit)
	_, err := deposit(closed, 1e9)
	assert.ErrorIs(t, err, ErrERC4626DepositMoreThanMax)
	_, err = closed.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: "0xvault", Amount: big.NewInt(1e9)}, TokenIn: "0xasset",
	})
	assert.ErrorIs(t, err, ErrERC4626DepositMoreThanMax)
	// redeems are not affected by a closed deposit side
	_, err = redeem(closed, 1e9)
	assert.NoError(t, err)

	// ignoreZeroMaxDeposit on the same reads keeps quoting deposits (totalAssets substitute)
	opened := sim(VaultCfg{Gas: GasCfg{Deposit: 80000, Redeem: 70000}, IgnoreZeroMaxDeposit: true})
	_, err = deposit(opened, 1e9)
	assert.NoError(t, err)
}

func TestRedeemsPaused(t *testing.T) {
	t.Parallel()
	sim := bothPool(t, 1, func(e *Extra) { e.RedeemsPaused = true })
	_, err := redeem(sim, 1e9)
	assert.ErrorIs(t, err, ErrERC4626RedeemsPaused)
	_, err = redeemIn(sim, 1e9)
	assert.ErrorIs(t, err, ErrERC4626RedeemsPaused)
	// the deposit side has its own flag
	_, err = deposit(sim, 1e9)
	assert.NoError(t, err)

	sim = bothPool(t, 1, func(e *Extra) { e.DepositsPaused = true })
	_, err = redeem(sim, 1e9)
	assert.NoError(t, err)
	_, err = redeemIn(sim, 1e9)
	assert.NoError(t, err)
	_, err = deposit(sim, 1e9)
	assert.ErrorIs(t, err, ErrERC4626DepositsPaused)
}

func TestRedeemMaxAgeStale(t *testing.T) {
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
			sim := bothPool(t, 1, func(e *Extra) { e.MaxAge = tc.maxAge })
			sim.poolTimestamp = now - tc.age
			_, err := redeem(sim, 1e9)
			_, errIn := redeemIn(sim, 1e9)
			if tc.wantStale {
				assert.ErrorIs(t, err, ErrERC4626StalePoolData)
				assert.ErrorIs(t, errIn, ErrERC4626StalePoolData)
			} else {
				assert.NoError(t, err)
				assert.NoError(t, errIn)
			}
		})
	}
}

// Indexing builds simulators without StaleCheck: an aged redeem-only pool must still quote there.
func TestRedeemFactoryStaleCheck(t *testing.T) {
	t.Parallel()
	for _, staleCheck := range []bool{false, true} {
		old := bothPool(t, 1, func(e *Extra) { e.MaxAge = 60 })
		b, err := json.Marshal(old.Extra)
		require.NoError(t, err)
		ep := entity.Pool{
			Address: "0xvault", Exchange: DexType, Type: DexType, Reserves: entity.PoolReserves{"0", "0"},
			Tokens:    []*entity.PoolToken{{Address: "0xvault", Swappable: true}, {Address: "0xasset", Swappable: true}},
			Timestamp: time.Now().Unix() - 3600, Extra: string(b),
		}
		iface, err := pool.Factory(DexType)(pool.FactoryParams{EntityPool: ep, Opts: pool.FactoryOpts{StaleCheck: staleCheck}})
		require.NoError(t, err)
		_, err = redeem(iface.(*PoolSimulator), 1e9)
		if staleCheck {
			assert.ErrorIs(t, err, ErrERC4626StalePoolData)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestUpdateEntityStateWritesRedeemsPaused(t *testing.T) {
	t.Parallel()
	z := big.NewInt(0)
	var p entity.Pool
	require.NoError(t, UpdateEntityState(&p, VaultCfg{Gas: GasCfg{Redeem: 1}},
		&PoolState{MaxDeposit: z, MaxRedeem: z, MinDeposit: z, TotalAssets: z, TotalSupply: z, RedeemsPaused: true}))
	assert.Contains(t, p.Extra, `"rP":true`)
	var e Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
	assert.True(t, e.RedeemsPaused)
}

// State saved before the field existed must decode as "not paused", and an unpaused vault must serialise
// exactly as before.
func TestExtraBackwardCompatible(t *testing.T) {
	t.Parallel()
	var e Extra
	require.NoError(t, json.Unmarshal([]byte(`{"g":{"d":1,"r":2},"mD":"0x5","p":true,"mA":120}`), &e))
	assert.False(t, e.RedeemsPaused)
	assert.True(t, e.DepositsPaused)
	assert.EqualValues(t, 120, e.MaxAge)

	b, err := json.Marshal(Extra{Gas: Gas{Deposit: 1}})
	require.NoError(t, err)
	assert.NotContains(t, string(b), "rP")

	// the config accepts the new keys and defaults them off
	var cfg VaultCfg
	require.NoError(t, json.Unmarshal([]byte(`{"gas":{"deposit":1}}`), &cfg))
	assert.Empty(t, cfg.RedeemsPausedMethod)
	require.NoError(t, json.Unmarshal([]byte(`{"gas":{"deposit":1},"redeemsPausedMethod":"paused"}`), &cfg))
	assert.Equal(t, "paused", cfg.RedeemsPausedMethod)
}

// A dust amount that rounds to zero output must be an error, as a zero-output swap is not a valid quote.
func TestZeroAmountOut(t *testing.T) {
	t.Parallel()
	// deposit: 2 assets per share, so 1 wei mints 0 shares. redeem: half an asset per share, so 1 wei yields 0.
	sim := bothPool(t, 2, func(*Extra) {})
	_, err := deposit(sim, 1)
	assert.ErrorIs(t, err, ErrZeroAmountOut)
	res, err := deposit(sim, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.TokenAmountOut.Amount.Int64())

	dust := bothPool(t, 1, func(e *Extra) {
		for i, a := range PrefetchAmounts {
			e.RedeemRates[i] = new(uint256.Int).Div(a, uint256.NewInt(2))
		}
	})
	_, err = redeem(dust, 1)
	assert.ErrorIs(t, err, ErrZeroAmountOut)
	res, err = redeem(dust, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.TokenAmountOut.Amount.Int64())
}
