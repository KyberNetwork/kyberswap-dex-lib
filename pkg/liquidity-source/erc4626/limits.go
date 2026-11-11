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

// AddLimitCalls adds the VaultV2 permission reads and the optional pause and supply-cap reads of a deposit-enabled
// vault. shareAddr is the contract whose totalSupply is the share supply. The supply lands in state.TotalSupply, so
// the redeem block must skip its own read when SupplyReadByLimits is true.
func AddLimitCalls(addFn func(*ethrpc.Call, []any), vaultAddr, shareAddr string, cfg VaultCfg, state *PoolState) {
	// a bool decodes as uint256 and stays nil when the vault has no such view
	for _, c := range []struct {
		method string
		out    **big.Int
	}{
		{VaultV2MethodCanReceiveShares, &state.CanReceiveShares},
		{VaultV2MethodCanSendShares, &state.CanSendShares},
		{VaultV2MethodCanSendAssets, &state.CanSendAssets},
	} {
		addFn(&ethrpc.Call{ABI: uint256OfAddressABI(c.method), Target: vaultAddr, Method: c.method,
			Params: []any{AddrDummy}}, []any{c.out})
	}
	if m := cfg.DepositsPausedMethod; m != "" {
		// a bool is one 0/1 word, so it decodes as uint256 (ethrpc cannot fill a nil *bool on a failed call)
		addFn(&ethrpc.Call{ABI: Uint256GetterABI(m), Target: vaultAddr, Method: m}, []any{&state.RawPaused})
	}
	if m := cfg.MaxSupplyMethod; m != "" {
		addFn(&ethrpc.Call{ABI: Uint256GetterABI(m), Target: vaultAddr, Method: m}, []any{&state.MaxSupply})
		addFn(&ethrpc.Call{ABI: ABI, Target: shareAddr, Method: Erc4626MethodTotalSupply}, []any{&state.TotalSupply})
	}
}

// ResolveMaxDeposit falls back to totalAssets when the maxDeposit read failed or its 0 does not mean closed: a
// Morpho VaultV2 open to any account (its maxDeposit is always 0 by design), or a vault configured with
// IgnoreZeroMaxDeposit. Any other 0 means deposits revert (cap reached, frozen, gated, ...).
func (s *PoolState) ResolveMaxDeposit(cfg VaultCfg) {
	if s.MaxDeposit == nil || (s.MaxDeposit.Sign() == 0 && (cfg.IgnoreZeroMaxDeposit || s.openVaultV2())) {
		s.MaxDeposit = s.TotalAssets
	} else if s.MaxDeposit.Cmp(bignumber.MaxUint128) > 0 {
		s.MaxDeposit = nil // no limit
	}
}

// openVaultV2 reports a VaultV2 that lets any account deposit and pass the shares on: deposit checks
// canReceiveShares(onBehalf) and canSendAssets(msg.sender), and the transfer out checks canSendShares.
func (s *PoolState) openVaultV2() bool {
	for _, v := range []*big.Int{s.CanReceiveShares, s.CanSendShares, s.CanSendAssets} {
		if v == nil || v.Sign() == 0 {
			return false
		}
	}
	return true
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
