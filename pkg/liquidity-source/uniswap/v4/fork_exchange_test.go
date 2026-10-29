package uniswapv4

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	abis "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var forkManager = common.HexToAddress("0x5fb015a49f5b35073f9ce01c2adb287a75f89600")

// A fork's pool must keep its own exchange key through every stage, or routing and tracking
// would attribute it to the official uniswap-v4 dex.
func TestForkExchange_UnhookedPoolKeepsDexID(t *testing.T) {
	cfg := &Config{ChainID: 1, DexID: "elyspump-v4", PoolManagerAddress: forkManager.Hex()}

	// factory stamping
	p := func() *entity.Pool {
		p, err := NewPoolFactory(cfg).newPool(&abis.UniswapV4PoolManagerInitialize{
			Id: [32]byte{1}, Fee: big.NewInt(3000), TickSpacing: big.NewInt(60),
			SqrtPriceX96: big.NewInt(1 << 62), Tick: big.NewInt(0),
		}, 1)
		require.NoError(t, err)
		return p
	}()
	require.Equal(t, "elyspump-v4", p.Exchange)
	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
	require.NotNil(t, se.PoolManager)
	assert.Equal(t, forkManager, *se.PoolManager)

	// simulator rebuild from the stored pool alone (no config)
	var stored entity.Pool
	require.NoError(t, json.Unmarshal([]byte(poolData), &stored))
	stored.Exchange, stored.StaticExtra = p.Exchange, p.StaticExtra
	sim, err := NewPoolSimulator(stored, 1)
	require.NoError(t, err)
	assert.Equal(t, "elyspump-v4", sim.GetExchange())

	// tracker update keeps it, with and without the tracker config
	for _, dexID := range []string{"elyspump-v4", ""} {
		tp := stored
		tr := &PoolTracker{}
		hp := &HookParam{Cfg: &Config{ChainID: 1, DexID: dexID}, Pool: &tp}
		require.NoError(t, tr.resolveHookState(context.Background(), &BaseHook{Exchange: DexType}, hp,
			&FetchRPCResult{Slot0: Slot0Data{SqrtPriceX96: big.NewInt(1 << 62), Tick: big.NewInt(0)}}, nil))
		assert.Equal(t, "elyspump-v4", tp.Exchange)
	}
}

// Existing pools and hook exchange names must stay byte-identical.
func TestForkExchange_DefaultUnchanged(t *testing.T) {
	for _, dexID := range []string{"", DexType} {
		for _, h := range []string{DexType, "uniswap-v4-fee", "uniswap-v4-passthru", "uniswap-v4-custom", "kem", "fairflow", "euler"} {
			assert.Equal(t, h, ForkExchange(h, dexID))
		}
	}
	assert.Equal(t, "elyspump-v4-fee", ForkExchange("uniswap-v4-fee", "elyspump-v4"))
	assert.Equal(t, "kem", ForkExchange("kem", "elyspump-v4"), "non-prefixed hook exchanges are left alone")

	var stored entity.Pool
	require.NoError(t, json.Unmarshal([]byte(poolData), &stored))
	sim, err := NewPoolSimulator(stored, 1)
	require.NoError(t, err)
	assert.Equal(t, DexType, sim.GetExchange())
	assert.Equal(t, valueobject.Exchange("uniswap-v4"), GetHookExchange(&stored))

	// hooked fork pool: stored suffix gives the base back; a stale "uniswap-v4" falls to official
	hook := &BaseHook{Exchange: "uniswap-v4-fee"}
	assert.Equal(t, "elyspump-v4-fee", ExchangeOf(hook, "", "elyspump-v4-fee"))
	assert.Equal(t, "elyspump-v4-fee", ExchangeOf(hook, "elyspump-v4", "elyspump-v4"))
	assert.Equal(t, "uniswap-v4-fee", ExchangeOf(hook, "", "uniswap-v4"))
}

// The encoder needs the pool's manager; old pools fall back to the per-chain map.
func TestPoolMetaInfo_PoolManager(t *testing.T) {
	var stored entity.Pool
	require.NoError(t, json.Unmarshal([]byte(poolData), &stored))

	meta := func(p entity.Pool) PoolMetaInfo {
		sim, err := NewPoolSimulator(p, valueobject.ChainIDEthereum)
		require.NoError(t, err)
		return sim.GetMetaInfo(stored.Tokens[0].Address, stored.Tokens[1].Address).(PoolMetaInfo)
	}
	m := meta(stored)
	require.NotNil(t, m.PoolManager)
	assert.Equal(t, PoolManager[valueobject.ChainIDEthereum], *m.PoolManager)

	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(stored.StaticExtra), &se))
	se.PoolManager = &forkManager
	b, _ := json.Marshal(se)
	stored.StaticExtra = string(b)
	m = meta(stored)
	require.NotNil(t, m.PoolManager)
	assert.Equal(t, forkManager, *m.PoolManager)

	// no manager known for the chain: field is omitted from JSON
	se.PoolManager = nil
	b, _ = json.Marshal(se)
	stored.StaticExtra = string(b)
	sim, err := NewPoolSimulator(stored, 99901)
	require.NoError(t, err)
	out, _ := json.Marshal(sim.GetMetaInfo(stored.Tokens[0].Address, stored.Tokens[1].Address))
	assert.NotContains(t, string(out), "manager")
}
