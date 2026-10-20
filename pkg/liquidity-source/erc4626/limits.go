package erc4626

import (
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/pkg/errors"
)

// SupplyReadByLimits reports whether AddLimitCalls already reads totalSupply into state.TotalSupply.
func (c VaultCfg) SupplyReadByLimits() bool {
	return c.Gas.Deposit > 0 && c.MaxSupplyMethod != ""
}

// AddLimitCalls adds the optional pause and supply-cap reads of a deposit-enabled vault.
// shareAddr is the contract whose totalSupply is the share supply. The supply lands in state.TotalSupply, so
// the redeem block must skip its own read when SupplyReadByLimits is true.
func AddLimitCalls(addFn func(*ethrpc.Call, []any), vaultAddr, shareAddr string, cfg VaultCfg, state *PoolState) {
	if m := cfg.DepositsPausedMethod; m != "" {
		// a bool is one 0/1 word, so it decodes as uint256 (ethrpc cannot fill a nil *bool on a failed call)
		addFn(&ethrpc.Call{ABI: Uint256GetterABI(m), Target: vaultAddr, Method: m}, []any{&state.RawPaused})
	}
	if m := cfg.MaxSupplyMethod; m != "" {
		addFn(&ethrpc.Call{ABI: Uint256GetterABI(m), Target: vaultAddr, Method: m}, []any{&state.MaxSupply})
		addFn(&ethrpc.Call{ABI: ABI, Target: shareAddr, Method: Erc4626MethodTotalSupply}, []any{&state.TotalSupply})
	}
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
