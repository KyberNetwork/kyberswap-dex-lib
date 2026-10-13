// Package tickermemev4 reuses Pons V2 pricing for Ticker's classic V4 pools.
// Tracking additionally verifies Ticker's upgradeable Monad deployment.
package tickermemev4

import (
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook struct {
	uniswapv4.BaseHook
	Extra
	decodeErr error
}

var _ = uniswapv4.RegisterHooksFactory(NewHook, HookAddress)

func NewHook(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{}
	if param.Cfg == nil || param.Cfg.ChainID != valueobject.ChainIDMonad || param.HookAddress != HookAddress {
		h.decodeErr = ErrUnsupportedDeployment
	} else if len(param.HookExtra) > 0 {
		// Do not retain partially decoded verification or fee state.
		var extra Extra
		if h.decodeErr = param.HookExtra.Unmarshal(&extra); h.decodeErr == nil {
			h.Extra = extra
		}
	}
	return h
}

func (h *Hook) GetExchange() string { return DexType }

func (h *Hook) validate() error {
	if h.decodeErr != nil {
		return h.decodeErr
	}
	if h.ImplementationHash != ImplementationCodeHash {
		return ErrUntracked
	}
	if h.Registered && (h.FeeBps < 0 || h.FeeBps > maxTradeFeeBps) {
		return ErrInvalidTerms
	}
	return nil
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if err := h.validate(); err != nil {
		return nil, err
	}
	result, err := h.Extra.Extra.AfterSwap(params)
	if err == nil && result.HookFee.Sign() > 0 {
		result.Gas = afterSwapGas
	}
	return result, err
}

func (h *Hook) CloneState() uniswapv4.Hook {
	clone := *h
	return &clone
}
