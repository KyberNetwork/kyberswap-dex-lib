package erc4626

import (
	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// A vault whose maxRedeem(owner) is min(balanceOf[owner], liquidity cap), e.g. stUSDS, reports 0 for the empty
// AddrDummy. The tracker gives AddrDummy a marker balance at each candidate balanceOf storage key and reads
// balanceOf back: a value 2^128+i means key i is the balance slot, so maxRedeem now returns the real cap.
// The slot is cached in StaticExtra, so later refreshes override that one key.

const (
	balanceMethod = "balanceOf"
	// ownerSlots is the number of plain mapping slots 0..N-1 probed, plus the OpenZeppelin v5 namespaced slot.
	ownerSlots = 21
	probeKeys  = ownerSlots + 1
)

var (
	// 2^128: above any real balance and low enough that maxWithdraw-style math (stUSDS overflows at 2^200) holds.
	markerBase = new(uint256.Int).Lsh(uint256.NewInt(1), 128)
	// ERC20StorageLocation of OpenZeppelin v5 ERC20, balances mapping at offset 0.
	ozV5ERC20Slot = common.HexToHash("0x52c63247e1f47db19d5ce0460030c497f067ca4cebf71ba98eeadabe20bace00")

	// balanceKeys[i] = keccak256(pad32(AddrDummy) . pad32(slot_i)), the storage key of balanceOf[AddrDummy].
	balanceKeys = func() (keys [probeKeys]common.Hash) {
		var slot [32]byte
		owner := common.LeftPadBytes(AddrDummy[:], 32)
		for i := range ownerSlots {
			slot[31] = byte(i)
			keys[i] = crypto.Keccak256Hash(owner, slot[:])
		}
		keys[ownerSlots] = crypto.Keccak256Hash(owner, ozV5ERC20Slot[:])
		return
	}()
)

// BalanceSlot is the cached probe outcome. The zero value means not probed yet.
type BalanceSlot struct {
	Key  common.Hash // storage key of balanceOf[AddrDummy]; zero if unknown
	None bool        // probed, no slot found; never probed again
}

func (b BalanceSlot) probing() bool { return !b.None }

// markerValue is the balance written at probe key i (2^128+i), or 2^128 for a cached key.
func markerValue(i int) common.Hash {
	var v uint256.Int
	v.SetUint64(uint64(i)).Add(&v, markerBase)
	return v.Bytes32()
}

// WithBalanceProbe adds the marker balances of AddrDummy on vaultAddr to the overrides: all candidate keys while the
// slot is unknown, only the cached key afterwards, nothing once probing found none.
func (c VaultCfg) WithBalanceProbe(vaultAddr string,
	overrides map[common.Address]gethclient.OverrideAccount) map[common.Address]gethclient.OverrideAccount {
	if c.Gas.Redeem == 0 || !c.Balance.probing() {
		return overrides
	}
	addr := common.HexToAddress(vaultAddr)
	out := make(map[common.Address]gethclient.OverrideAccount, len(overrides)+1)
	for a, o := range overrides {
		out[a] = o
	}
	o := out[addr]
	diff := make(map[common.Hash]common.Hash, len(o.StateDiff)+probeKeys)
	for k, v := range o.StateDiff {
		diff[k] = v
	}
	if c.Balance.Key != (common.Hash{}) {
		diff[c.Balance.Key] = markerValue(0)
	} else {
		for i, k := range balanceKeys {
			diff[k] = markerValue(i)
		}
	}
	o.StateDiff = diff
	out[addr] = o
	return out
}

// AddBalanceProbeCalls adds the balanceOf(AddrDummy) read-back, and the VaultV2 permission reads that tell a
// by-design maxRedeem 0 apart when AddLimitCalls does not run (deposit disabled).
func AddBalanceProbeCalls(addFn func(*ethrpc.Call, []any), vaultAddr string, cfg VaultCfg, state *PoolState) {
	if cfg.Gas.Redeem == 0 || !cfg.Balance.probing() {
		return
	}
	addFn(&ethrpc.Call{ABI: uint256OfAddressABI(balanceMethod), Target: vaultAddr, Method: balanceMethod,
		Params: []any{AddrDummy}}, []any{&state.BalanceOf})
	if cfg.Gas.Deposit == 0 {
		addFn(&ethrpc.Call{ABI: uint256OfAddressABI(VaultV2MethodCanSendShares), Target: vaultAddr,
			Method: VaultV2MethodCanSendShares, Params: []any{AddrDummy}}, []any{&state.CanSendShares})
	}
}

// resolveBalanceProbe reads the balanceOf read-back: a marker hit means maxRedeem is the real cap. It records the
// slot to persist in Slot (SlotChanged set) when the probe found it, found none, or the cached key stopped working.
// A missing read-back (no probe, lazy batch path, failed call) changes nothing.
func (s *PoolState) resolveBalanceProbe(cfg VaultCfg) {
	if s.BalanceOf == nil || cfg.Gas.Redeem == 0 || !cfg.Balance.probing() {
		return
	}
	s.Slot = cfg.Balance
	bal, ovf := uint256.FromBig(s.BalanceOf)
	over := ovf || bal.Lt(markerBase) // not a marker
	bal.Sub(bal, markerBase)
	switch {
	case cfg.Balance.Key != (common.Hash{}): // cached key: hit while the read-back is exactly 2^128
		if s.CapKnown = !over && bal.IsZero(); !s.CapKnown {
			s.Slot = BalanceSlot{} // vault changed: probe again next refresh
		}
	case !over && bal.LtUint64(probeKeys):
		s.CapKnown = true
		s.Slot.Key = balanceKeys[bal.Uint64()]
	default:
		s.Slot.None = true
	}
	s.SlotChanged = s.Slot != cfg.Balance
}

// ResolveMaxRedeem turns the raw maxRedeem read into the redeem limit (nil = unlimited).
//   - With a marker hit maxRedeem is the real cap: 0 stays closed, otherwise min(cap, totalSupply). The exception is
//     a Morpho VaultV2 (its permission views exist), whose maxRedeem is 0 by design while redeem works.
//   - Otherwise a 0 or failed read falls back to totalSupply and a value above MaxUint128 means no limit.
func (s *PoolState) ResolveMaxRedeem(cfg VaultCfg) {
	s.resolveBalanceProbe(cfg)
	switch {
	case cfg.Gas.Redeem == 0: // redeem unsupported: totalSupply is read only for the deposit conversion
		s.MaxRedeem = nil
	case s.CapKnown && s.MaxRedeem != nil && (s.MaxRedeem.Sign() != 0 || s.CanSendShares == nil):
		if s.MaxRedeem.Cmp(bignumber.MaxUint128) > 0 {
			s.MaxRedeem = s.TotalSupply // balance-bound, not a cap
		} else if s.TotalSupply != nil && s.TotalSupply.Cmp(s.MaxRedeem) < 0 {
			s.MaxRedeem = s.TotalSupply
		}
	case s.MaxRedeem == nil || s.MaxRedeem.Sign() == 0:
		s.MaxRedeem = s.TotalSupply // fallback to a sensible value
	case s.MaxRedeem.Cmp(bignumber.MaxUint128) > 0:
		s.MaxRedeem = nil // no limit
	}
}

// LoadBalanceSlot reads the cached cap-probe outcome from the pool's StaticExtra.
func LoadBalanceSlot(p *entity.Pool) BalanceSlot {
	var se StaticExtra
	_ = json.Unmarshal([]byte(p.StaticExtra), &se)
	return se.BalanceSlot()
}

// SaveBalanceSlot writes a changed probe outcome into the pool's StaticExtra, keeping its other fields.
func SaveBalanceSlot(p *entity.Pool, state *PoolState) error {
	if !state.SlotChanged {
		return nil
	}
	var se StaticExtra
	if p.StaticExtra != "" {
		if err := json.Unmarshal([]byte(p.StaticExtra), &se); err != nil {
			return err
		}
	}
	se.BalanceKey, se.NoBalanceKey = "", state.Slot.None
	if state.Slot.Key != (common.Hash{}) {
		se.BalanceKey = state.Slot.Key.Hex()
	}
	b, err := json.Marshal(se)
	if err != nil {
		return err
	}
	p.StaticExtra = string(b)
	return nil
}

// NeedsCleanRead reports a probe run that ended without a cached key (none found, or the cached key stopped
// working). Its markers may alias unrelated storage, e.g. Euler packs balance and flags in one slot so a marker
// reads back as balance 13, which would end up as maxRedeem. FetchAssetAndState then reads again without
// overrides; the vault is not probed again, so this happens once per vault.
func (s *PoolState) NeedsCleanRead() bool {
	return s.SlotChanged && s.Slot.Key == (common.Hash{})
}
