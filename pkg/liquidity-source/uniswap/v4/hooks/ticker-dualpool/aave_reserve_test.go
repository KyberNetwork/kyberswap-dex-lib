package tickerdualpool

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestAaveReserveCashShareCashRounding(t *testing.T) {
	r := AaveReserve{Cash: uint256.NewInt(1000), ScaledSupply: uint256.NewInt(10000), SupplyCap: uint256.NewInt(0), Active: true}
	rate := new(uint256.Int).Add(ray, uint256.NewInt(1))
	limit, err := r.withdrawalLimit(uint256.NewInt(2000), rate)
	require.NoError(t, err)
	require.Equal(t, "999", limit.Dec())
	require.ErrorIs(t, r.withdraw(uint256.NewInt(1000), uint256.NewInt(2000), rate), ErrVaultIlliquid)
	require.Equal(t, "1000", r.Cash.Dec(), "failed withdrawal must not consume cash")
}

func TestAaveReserveWithdrawalFreesCapButDoesNotUnfreeze(t *testing.T) {
	for _, frozen := range []bool{false, true} {
		r := AaveReserve{Cash: uint256.NewInt(1000), ScaledSupply: uint256.NewInt(1000), SupplyCap: uint256.NewInt(1000), Active: true, Frozen: frozen}
		limit, err := r.depositLimit(ray)
		require.NoError(t, err)
		require.True(t, limit.IsZero())
		require.NoError(t, r.withdraw(uint256.NewInt(400), uint256.NewInt(1000), ray))
		accepted, err := r.deposit(uint256.NewInt(400), ray)
		require.NoError(t, err)
		require.Equal(t, !frozen, accepted)
		if frozen {
			require.Equal(t, "600", r.Cash.Dec())
		} else {
			require.Equal(t, "1000", r.Cash.Dec())
		}
	}
}

func TestAaveFrozenRedepositKeepsPoolCash(t *testing.T) {
	v := AaveInventory{Raw: uint256.NewInt(0), Claims: uint256.NewInt(0), PoolShares: uint256.NewInt(1000000000), Supply: uint256.NewInt(1000000000), Idle: uint256.NewInt(0), StrategyShares: uint256.NewInt(1000), Rate: ray.Clone(), PendingYield: uint256.NewInt(0)}
	reserve := AaveReserve{Cash: uint256.NewInt(1000), ScaledSupply: uint256.NewInt(1000), SupplyCap: uint256.NewInt(0), Active: true, Frozen: true}
	next, r, err := v.AfterJITWithReserve(uint256.NewInt(400), uint256.NewInt(300), reserve)
	require.NoError(t, err)
	require.Equal(t, "300", next.Raw.Dec(), "withdraw 400, spend net 100, keep 300 when deposit fails")
	require.Equal(t, "600", next.StrategyShares.Dec())
	require.Equal(t, "600", r.Cash.Dec())
	require.Equal(t, "1000", reserve.Cash.Dec(), "quote must not mutate input reserve")
	require.Equal(t, "1000", v.StrategyShares.Dec())
	reserve.Frozen = false
	next, r, err = v.AfterJITWithReserve(uint256.NewInt(400), uint256.NewInt(300), reserve)
	require.NoError(t, err)
	require.True(t, next.Raw.IsZero())
	require.Equal(t, "900", next.StrategyShares.Dec())
	require.Equal(t, "900", r.Cash.Dec())
}

func TestAavePausedWithdrawalRejected(t *testing.T) {
	r := AaveReserve{Cash: uint256.NewInt(1000), ScaledSupply: uint256.NewInt(1000), SupplyCap: uint256.NewInt(0), Active: true, Paused: true}
	require.ErrorIs(t, r.withdraw(uint256.NewInt(1), uint256.NewInt(1000), ray), ErrVaultIlliquid)
	accepted, err := r.deposit(uint256.NewInt(1), ray)
	require.NoError(t, err)
	require.False(t, accepted)
}

func TestTreasuryAccrualCanExhaustDepositCap(t *testing.T) {
	r := AaveReserve{Cash: uint256.NewInt(990), ScaledSupply: uint256.NewInt(990), SupplyCap: uint256.NewInt(1000), PendingTreasury: uint256.NewInt(10), Active: true}
	limit, err := r.depositLimit(ray)
	require.NoError(t, err)
	require.Equal(t, "10", limit.Dec(), "maxDeposit sees stored treasury before the reserve update")
	accepted, err := r.deposit(uint256.NewInt(10), ray)
	require.NoError(t, err)
	require.False(t, accepted, "pool validation sees newly accrued treasury and rejects the deposit")
	require.Equal(t, "990", r.ScaledSupply.Dec(), "reverted deposit must not accrue or mint")
	require.Equal(t, "10", r.PendingTreasury.Dec())
	require.NoError(t, r.withdraw(uint256.NewInt(20), uint256.NewInt(990), ray))
	require.Equal(t, "980", r.ScaledSupply.Dec(), "withdrawal burns 20 and accrues 10")
	require.True(t, r.PendingTreasury.IsZero())
	limit, err = r.depositLimit(ray)
	require.NoError(t, err)
	require.Equal(t, "20", limit.Dec())
}

func TestElapsedTreasuryAccrualMatchesMonadFork(t *testing.T) {
	raw := []byte(capturedAaveAccrual)
	var f map[string]string
	require.NoError(t, json.Unmarshal(raw, &f))
	n := func(key string) *uint256.Int { return uint256.MustFromDecimal(f[key]) }
	factor, err := strconv.ParseUint(f["factor"], 10, 64)
	require.NoError(t, err)
	pending, err := pendingTreasuryShares(n("scaledDebt"), n("previousIndex"), n("nextIndex"), n("rate"), factor)
	require.NoError(t, err)
	require.Equal(t, f["treasuryDelta"], pending.Dec())
	reserve := AaveReserve{Cash: n("cashBefore"), ScaledSupply: n("supplyBefore"), SupplyCap: new(uint256.Int), PendingTreasury: pending, Active: true}
	require.NoError(t, reserve.withdraw(n("withdraw"), n("held"), n("rate")))
	require.Equal(t, f["supplyAfter"], reserve.ScaledSupply.Dec())
	require.Equal(t, f["cashAfter"], reserve.Cash.Dec())
	require.True(t, reserve.PendingTreasury.IsZero())
}

// Captured contract results; preserved from aave-accrual.json.
const capturedAaveAccrual = `{
"rate":"1010239873045829464919591085",
"factor":"1000",
"scaledDebt":"156522557133680",
"previousIndex":"1012526855659549145199017959",
"nextIndex":"1012664406964246121576712621",
"supplyBefore":"171976086350974",
"cashBefore":"15234432117736",
"held":"9899743234",
"withdraw":"100000000",
"treasuryDelta":"2131165332",
"supplyAfter":"171978118529914",
"cashAfter":"15234332117736"
}`
