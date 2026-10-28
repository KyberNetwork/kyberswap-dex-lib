package lazy

import (
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
// unsupported, so normalizePoolState must not turn the nil maxRedeem into totalSupply.
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
	normalizePoolState(cfg, st)
	assert.Nil(t, st.MaxRedeem)
}
