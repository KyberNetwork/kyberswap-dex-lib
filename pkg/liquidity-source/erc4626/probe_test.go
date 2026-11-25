package erc4626

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

var probeCfg = VaultCfg{Gas: GasCfg{Deposit: 1, Redeem: 1}}

// marker returns the balanceOf(AddrDummy) read-back when key i holds the balance.
func marker(i int64) *big.Int {
	return new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(i))
}

func TestProbeKeys(t *testing.T) {
	t.Parallel()
	// slot 2 is stUSDS: key = keccak(pad32(dummy) . pad32(2)), the Solidity mapping layout
	want := crypto.Keccak256Hash(common.LeftPadBytes(AddrDummy[:], 32), common.LeftPadBytes([]byte{2}, 32))
	require.Equal(t, want, balanceKeys[2])
	require.Equal(t, crypto.Keccak256Hash(common.LeftPadBytes(AddrDummy[:], 32), ozV5ERC20Slot[:]), balanceKeys[ownerSlots])
	// the OZ v5 constant is keccak(abi.encode(uint256(keccak256("openzeppelin.storage.ERC20")) - 1)) & ~0xff
	id := new(big.Int).SetBytes(crypto.Keccak256([]byte("openzeppelin.storage.ERC20")))
	id.Sub(id, big.NewInt(1))
	h := crypto.Keccak256(common.LeftPadBytes(id.Bytes(), 32))
	h[31] = 0
	require.Equal(t, common.BytesToHash(h), ozV5ERC20Slot)
	seen := map[common.Hash]bool{}
	for _, k := range balanceKeys {
		seen[k] = true
	}
	require.Len(t, seen, probeKeys)
}

func TestWithBalanceProbeOverrides(t *testing.T) {
	t.Parallel()
	vault := "0x99CD4Ec3f88A45940936F469E4bB72A2A701EEB9"
	addr := common.HexToAddress(vault)

	diff := probeCfg.WithBalanceProbe(vault, nil)[addr].StateDiff
	require.Len(t, diff, probeKeys)
	for i, k := range balanceKeys { // each key carries its own marker 2^128+i
		require.Equal(t, markerValue(i), diff[k])
		require.Equal(t, marker(int64(i)), diff[k].Big())
	}

	cached := probeCfg
	cached.Balance = BalanceSlot{Key: balanceKeys[2]}
	diff = cached.WithBalanceProbe(vault, nil)[addr].StateDiff
	require.Len(t, diff, 1) // cached slot: a single override of 2^128
	require.Equal(t, marker(0), diff[balanceKeys[2]].Big())

	none := probeCfg
	none.Balance.None = true
	require.Nil(t, none.WithBalanceProbe(vault, nil))
	require.Nil(t, VaultCfg{Gas: GasCfg{Deposit: 1}}.WithBalanceProbe(vault, nil)) // redeem disabled
}

func TestResolveMaxRedeem(t *testing.T) {
	t.Parallel()
	ts := big.NewInt(149)
	huge := new(big.Int).Lsh(big.NewInt(1), 130)
	cached := probeCfg
	cached.Balance = BalanceSlot{Key: balanceKeys[2]}
	noRedeem := VaultCfg{Gas: GasCfg{Deposit: 1}}

	for _, tc := range []struct {
		name  string
		cfg   VaultCfg
		state PoolState
		want  *big.Int // nil = unlimited
	}{
		{"stUSDS cap 0 stays closed", probeCfg, PoolState{BalanceOf: marker(2), MaxRedeem: new(big.Int), TotalSupply: ts}, big.NewInt(0)},
		{"cached slot cap 0 stays closed", cached, PoolState{BalanceOf: marker(0), MaxRedeem: new(big.Int), TotalSupply: ts}, big.NewInt(0)},
		{"cap kept", probeCfg, PoolState{BalanceOf: marker(2), MaxRedeem: big.NewInt(7), TotalSupply: ts}, big.NewInt(7)},
		{"cap min totalSupply", probeCfg, PoolState{BalanceOf: marker(2), MaxRedeem: big.NewInt(500), TotalSupply: ts}, ts},
		{"hit, balance-bound maxRedeem above MaxUint128 uses totalSupply", probeCfg,
			PoolState{BalanceOf: marker(2), MaxRedeem: huge, TotalSupply: ts}, ts},
		{"hit, failed maxRedeem read falls back", probeCfg, PoolState{BalanceOf: marker(2), TotalSupply: ts}, ts},
		{"no marker 0 falls back to totalSupply", probeCfg, PoolState{BalanceOf: big.NewInt(0), MaxRedeem: new(big.Int), TotalSupply: ts}, ts},
		{"no probe (lazy batch) 0 falls back", probeCfg, PoolState{MaxRedeem: new(big.Int), TotalSupply: ts}, ts},
		{"cached slot but no read-back (lazy batch) never closes", cached, PoolState{MaxRedeem: new(big.Int), TotalSupply: ts}, ts},
		{"cached slot stopped working falls back", cached, PoolState{BalanceOf: big.NewInt(0), MaxRedeem: new(big.Int), TotalSupply: ts}, ts},
		{"VaultV2 0 by design falls back", probeCfg, PoolState{BalanceOf: marker(12), MaxRedeem: new(big.Int),
			CanSendShares: big.NewInt(1), TotalSupply: ts}, ts},
		{"VaultV2 nonzero cap kept", probeCfg, PoolState{BalanceOf: marker(12), MaxRedeem: big.NewInt(5),
			CanSendShares: big.NewInt(0), TotalSupply: ts}, big.NewInt(5)},
		{"no marker huge maxRedeem is unlimited", probeCfg, PoolState{MaxRedeem: huge, TotalSupply: ts}, nil},
		{"no marker maxRedeem kept", probeCfg, PoolState{MaxRedeem: big.NewInt(9), TotalSupply: ts}, big.NewInt(9)},
		{"redeem unsupported", noRedeem, PoolState{BalanceOf: marker(2), MaxRedeem: big.NewInt(9), TotalSupply: ts}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.state
			s.ResolveMaxRedeem(tc.cfg)
			if tc.want == nil {
				require.Nil(t, s.MaxRedeem)
			} else {
				require.Zero(t, tc.want.Cmp(s.MaxRedeem), "got %v", s.MaxRedeem)
			}
		})
	}
}

func TestResolveBalanceProbeSlot(t *testing.T) {
	t.Parallel()
	ts := big.NewInt(1)
	run := func(cfg VaultCfg, bal *big.Int) *PoolState {
		s := &PoolState{BalanceOf: bal, TotalSupply: ts}
		s.ResolveMaxRedeem(cfg)
		return s
	}
	s := run(probeCfg, marker(2))
	require.True(t, s.SlotChanged)
	require.Equal(t, BalanceSlot{Key: balanceKeys[2]}, s.Slot)

	s = run(probeCfg, marker(probeKeys)) // one past the last marker
	require.Equal(t, BalanceSlot{None: true}, s.Slot)
	s = run(probeCfg, big.NewInt(0))
	require.Equal(t, BalanceSlot{None: true}, s.Slot)
	require.False(t, s.CapKnown)

	cached := probeCfg
	cached.Balance = BalanceSlot{Key: balanceKeys[2]}
	s = run(cached, marker(0))
	require.False(t, s.SlotChanged) // still valid: nothing to write
	s = run(cached, big.NewInt(0))
	require.True(t, s.SlotChanged)
	require.Equal(t, BalanceSlot{}, s.Slot) // reset, probed again next refresh

	s = run(probeCfg, nil) // failed read-back or lazy batch: nothing recorded
	require.False(t, s.SlotChanged)
}

func TestStaticExtraRoundTrip(t *testing.T) {
	t.Parallel()
	p := &entity.Pool{StaticExtra: `{"isNativeAsset":true}`}
	require.NoError(t, SaveBalanceSlot(p, &PoolState{SlotChanged: true, Slot: BalanceSlot{Key: balanceKeys[2]}}))
	require.Equal(t, BalanceSlot{Key: balanceKeys[2]}, LoadBalanceSlot(p))
	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
	require.True(t, se.IsNativeAsset) // kept

	require.NoError(t, SaveBalanceSlot(p, &PoolState{SlotChanged: true, Slot: BalanceSlot{None: true}}))
	require.Equal(t, BalanceSlot{None: true}, LoadBalanceSlot(p))
	require.NoError(t, SaveBalanceSlot(p, &PoolState{SlotChanged: true}))
	require.Equal(t, BalanceSlot{}, LoadBalanceSlot(p))

	before := p.StaticExtra
	require.NoError(t, SaveBalanceSlot(p, &PoolState{}))
	require.Equal(t, before, p.StaticExtra) // unchanged probe leaves StaticExtra alone

	empty := &entity.Pool{}
	require.NoError(t, SaveBalanceSlot(empty, &PoolState{SlotChanged: true, Slot: BalanceSlot{None: true}}))
	require.Equal(t, `{"nbK":true}`, empty.StaticExtra)
}

// Euler packs the balance with flags in one slot: a marker at its mapping slot reads back as balance i (here 13)
// and would become maxRedeem. Such a probe run must be re-read without overrides.
func TestNeedsCleanRead(t *testing.T) {
	t.Parallel()
	s := PoolState{BalanceOf: big.NewInt(13), MaxRedeem: big.NewInt(13), TotalSupply: big.NewInt(149)}
	s.ResolveMaxRedeem(probeCfg)
	require.True(t, s.NeedsCleanRead())

	hit := PoolState{BalanceOf: marker(0), MaxRedeem: big.NewInt(5), TotalSupply: big.NewInt(149)}
	hit.ResolveMaxRedeem(probeCfg)
	require.False(t, hit.NeedsCleanRead()) // marker hit: the cap is genuine

	cached := probeCfg
	cached.Balance = BalanceSlot{Key: balanceKeys[2]}
	stale := PoolState{BalanceOf: big.NewInt(0), TotalSupply: big.NewInt(149)}
	stale.ResolveMaxRedeem(cached)
	require.True(t, stale.NeedsCleanRead())

	ok := PoolState{BalanceOf: marker(0), MaxRedeem: big.NewInt(5), TotalSupply: big.NewInt(149)}
	ok.ResolveMaxRedeem(cached)
	require.False(t, ok.NeedsCleanRead())
	require.False(t, (&PoolState{MaxRedeem: big.NewInt(5)}).NeedsCleanRead()) // no probe (lazy batch)
}
