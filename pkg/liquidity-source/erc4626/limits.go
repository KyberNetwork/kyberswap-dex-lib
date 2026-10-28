package erc4626

import (
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// SupplyReadByLimits reports whether AddLimitCalls already reads totalSupply into state.TotalSupply.
func (c VaultCfg) SupplyReadByLimits() bool {
	return c.Gas.Deposit > 0 && c.MaxSupplyMethod != ""
}

// AddLimitCalls adds the VaultV2 gate reads and the optional pause and supply-cap reads of a deposit-enabled vault.
// shareAddr is the contract whose totalSupply is the share supply. The supply lands in state.TotalSupply, so
// the redeem block must skip its own read when SupplyReadByLimits is true.
func AddLimitCalls(addFn func(*ethrpc.Call, []any), vaultAddr, shareAddr string, cfg VaultCfg, state *PoolState) {
	// an address is one word, so it decodes as uint256 and stays nil when the getter is missing
	addFn(&ethrpc.Call{ABI: Uint256GetterABI(VaultV2MethodReceiveSharesGate), Target: vaultAddr,
		Method: VaultV2MethodReceiveSharesGate}, []any{&state.ReceiveSharesGate})
	addFn(&ethrpc.Call{ABI: Uint256GetterABI(VaultV2MethodSendAssetsGate), Target: vaultAddr,
		Method: VaultV2MethodSendAssetsGate}, []any{&state.SendAssetsGate})
	if m := cfg.DepositsPausedMethod; m != "" {
		// a bool is one 0/1 word, so it decodes as uint256 (ethrpc cannot fill a nil *bool on a failed call)
		addFn(&ethrpc.Call{ABI: Uint256GetterABI(m), Target: vaultAddr, Method: m}, []any{&state.RawPaused})
	}
	if m := cfg.MaxSupplyMethod; m != "" {
		addFn(&ethrpc.Call{ABI: Uint256GetterABI(m), Target: vaultAddr, Method: m}, []any{&state.MaxSupply})
		addFn(&ethrpc.Call{ABI: ABI, Target: shareAddr, Method: Erc4626MethodTotalSupply}, []any{&state.TotalSupply})
	}
}

// ResolveMaxDeposit falls back to totalAssets when the maxDeposit read failed or the vault is an ungated Morpho
// VaultV2, whose maxDeposit is always 0 by design. Any other 0 means deposits revert (cap reached, frozen, ...).
func (s *PoolState) ResolveMaxDeposit() {
	if s.MaxDeposit == nil || (s.MaxDeposit.Sign() == 0 && s.ungatedVaultV2()) {
		s.MaxDeposit = s.TotalAssets
	} else if s.MaxDeposit.Cmp(bignumber.MaxUint128) > 0 {
		s.MaxDeposit = nil // no limit
	}
}

// ungatedVaultV2 reports a VaultV2 with no gate on deposits: its deposit checks canReceiveShares(onBehalf) and
// canSendAssets(msg.sender), which only a set gate can refuse.
func (s *PoolState) ungatedVaultV2() bool {
	return s.ReceiveSharesGate != nil && s.SendAssetsGate != nil &&
		s.ReceiveSharesGate.Sign() == 0 && s.SendAssetsGate.Sign() == 0
}

// ResolveLimits derives Paused and MintHeadroom from the raw reads. A configured read that failed is an error,
// so a revert cannot pass as "not paused" or "no cap".
func (s *PoolState) ResolveLimits(cfg VaultCfg) error {
	if cfg.Gas.Deposit == 0 {
		return nil
	}
	if cfg.DepositsPausedMethod != "" {
		if s.RawPaused == nil {
			return errors.Wrap(ErrReadFailed, cfg.DepositsPausedMethod)
		}
		s.Paused = s.RawPaused.Sign() != 0
	}
	if cfg.MaxSupplyMethod != "" {
		if s.MaxSupply == nil || s.TotalSupply == nil {
			return errors.Wrap(ErrReadFailed, cfg.MaxSupplyMethod)
		}
		s.MintHeadroom = new(big.Int)
		if s.MaxSupply.Cmp(s.TotalSupply) > 0 {
			s.MintHeadroom.Sub(s.MaxSupply, s.TotalSupply)
		}
	}
	return nil
}
