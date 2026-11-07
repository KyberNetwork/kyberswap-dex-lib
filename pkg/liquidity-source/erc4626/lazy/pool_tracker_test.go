package lazy

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	erc4626 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
)

// The lazy tracker is the primary one for erc4626 pools, so it must read the pause and cap getters too.
func TestAddStateCallsReadsLimits(t *testing.T) {
	t.Parallel()
	var methods []string
	add := func(c *ethrpc.Call, _ []any) { methods = append(methods, c.Method) }
	cfg := erc4626.VaultCfg{Gas: erc4626.GasCfg{Deposit: 1}, DepositsPausedMethod: "depositsPaused", MaxSupplyMethod: "maxSupply"}

	addStateCalls(add, "0xvault", "", cfg, false, new(common.Address), newState())
	assert.Contains(t, methods, "depositsPaused")
	assert.Contains(t, methods, "maxSupply")

	methods = nil
	cfg.DepositsPausedMethod, cfg.MaxSupplyMethod = "", ""
	addStateCalls(add, "0xvault", "", cfg, false, new(common.Address), newState())
	assert.NotContains(t, methods, "depositsPaused")
	assert.NotContains(t, methods, "maxSupply")
}

func newState() *erc4626.PoolState {
	return &erc4626.PoolState{
		DepositRates: make([]*big.Int, len(erc4626.PrefetchAmounts)),
		RedeemRates:  make([]*big.Int, len(erc4626.PrefetchAmounts)),
	}
}

func TestUpdateEntityStateKeepsLimits(t *testing.T) {
	t.Parallel()
	var p entity.Pool
	st := newState()
	st.MaxDeposit, st.MaxRedeem, st.MinDeposit, st.TotalAssets = big.NewInt(5), big.NewInt(0), big.NewInt(0), big.NewInt(7)
	st.Paused, st.MintHeadroom = true, big.NewInt(0)
	require.NoError(t, UpdateEntityState(&p, erc4626.VaultCfg{Gas: erc4626.GasCfg{Deposit: 1}}, st))

	var e erc4626.Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
	assert.True(t, e.DepositsPaused)
	require.NotNil(t, e.MintHeadroom)
	assert.Zero(t, e.MintHeadroom.Sign())
}

// A deposit-only vault must still read totalSupply: slot 0 converts assets to shares with it. Redeem stays
// unsupported, so NormalizeState must not turn the nil maxRedeem into totalSupply.
func TestDepositOnlyReadsSupplyKeepsRedeemOff(t *testing.T) {
	t.Parallel()
	var methods []string
	add := func(c *ethrpc.Call, _ []any) { methods = append(methods, c.Method) }
	cfg := erc4626.VaultCfg{Gas: erc4626.GasCfg{Deposit: 1}}
	addStateCalls(add, "0xvault", "", cfg, false, new(common.Address), newState())
	assert.Contains(t, methods, erc4626.Erc4626MethodTotalSupply)
	assert.NotContains(t, methods, erc4626.Erc4626MethodMaxRedeem)

	st := newState()
	st.TotalSupply, st.TotalAssets = big.NewInt(5), big.NewInt(7)
	erc4626.NormalizeState(cfg, st)
	assert.Nil(t, st.MaxRedeem)
}

// Redeem pause is read for redeem-enabled vaults only, with or without deposits, and never for deposit-only.
func TestAddStateCallsReadsRedeemsPaused(t *testing.T) {
	t.Parallel()
	collect := func(cfg erc4626.VaultCfg) (methods []string) {
		add := func(c *ethrpc.Call, _ []any) { methods = append(methods, c.Method) }
		addStateCalls(add, "0xvault", "", cfg, false, new(common.Address), newState())
		return
	}
	redeemOnly := erc4626.VaultCfg{Gas: erc4626.GasCfg{Redeem: 1}, RedeemsPausedMethod: "paused"}
	assert.Contains(t, collect(redeemOnly), "paused")

	both := erc4626.VaultCfg{Gas: erc4626.GasCfg{Deposit: 1, Redeem: 1}, DepositsPausedMethod: "paused", RedeemsPausedMethod: "paused"}
	assert.Equal(t, 2, countOf(collect(both), "paused"))

	both.RedeemsPausedMethod = ""
	assert.Equal(t, 1, countOf(collect(both), "paused"))

	depositOnly := erc4626.VaultCfg{Gas: erc4626.GasCfg{Deposit: 1}, RedeemsPausedMethod: "paused"}
	assert.NotContains(t, collect(depositOnly), "paused")
}

func countOf(ss []string, s string) (n int) {
	for _, v := range ss {
		if v == s {
			n++
		}
	}
	return
}

func TestUpdateEntityStateWritesRedeemsPaused(t *testing.T) {
	t.Parallel()
	var p entity.Pool
	st := newState()
	st.MaxDeposit, st.MaxRedeem, st.MinDeposit, st.TotalAssets = big.NewInt(5), big.NewInt(5), big.NewInt(0), big.NewInt(7)
	st.RedeemsPaused = true
	require.NoError(t, UpdateEntityState(&p, erc4626.VaultCfg{Gas: erc4626.GasCfg{Redeem: 1}}, st))

	var e erc4626.Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
	assert.True(t, e.RedeemsPaused)
	assert.False(t, e.DepositsPaused)
}

// A configured redeem pause read that failed fails the lazy refresh, so a revert cannot pass as "not paused".
func TestLazycallRedeemsPausedFailsOnMissingRead(t *testing.T) {
	t.Parallel()
	p := entity.Pool{Address: "0xvault"}
	cfg := erc4626.VaultCfg{Gas: erc4626.GasCfg{Redeem: 1}, RedeemsPausedMethod: "paused"}
	_, apply := Lazycall(context.Background(), &p, ethrpc.New("http://127.0.0.1:1"), "0xvault", "", cfg, false, nil)
	_, err := apply(big.NewInt(1))
	assert.ErrorIs(t, err, erc4626.ErrReadFailed)
}
