// Package tickermemev4 models Ticker's separately rounded V4 output fees.
// Tracking additionally verifies Ticker's upgradeable Monad deployment.
package tickermemev4

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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
	if h.Registered && uint32(h.HookFeeBps)+uint32(h.CreatorTaxBps) > maxTradeFeeBps {
		return ErrInvalidTerms
	}
	return nil
}

// Enable the simulator callback to gross up reverse quotes; the contract has no beforeSwap callback.
func (h *Hook) CanBeforeSwap(common.Address) bool { return true }

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if err := h.validate(); err != nil {
		return nil, err
	}
	if params.CalcOut || !h.Registered {
		return h.BaseHook.BeforeSwap(params)
	}
	net := params.AmountSpecified
	if net == nil || net.Sign() <= 0 || net.BitLen() > 127 {
		return nil, uniswapv4.ErrInvalidAmountOut
	}
	rate := int64(h.HookFeeBps) + int64(h.CreatorTaxBps)
	gross := uniswapv4.GrossBeforeFee(net, big.NewInt(rate), big.NewInt(basisPoints))
	for {
		fee, err := h.fee(gross)
		if err != nil {
			return nil, err
		}
		if new(big.Int).Sub(gross, fee).Cmp(net) >= 0 {
			result := &uniswapv4.BeforeSwapResult{
				DeltaSpecified:   new(big.Int).Sub(gross, net),
				DeltaUnspecified: new(big.Int),
			}
			if fee.Sign() > 0 {
				result.Gas = afterSwapGas
			}
			return result, nil
		}
		gross.Add(gross, bignumber.One)
	}
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if err := h.validate(); err != nil {
		return nil, err
	}
	if !params.CalcOut {
		// BeforeSwap already included the output fees in the requested gross output.
		return h.BaseHook.AfterSwap(params)
	}
	fee, err := h.fee(params.AmountOut)
	if err != nil {
		return nil, err
	}
	result := &uniswapv4.AfterSwapResult{HookFee: fee}
	if fee.Sign() > 0 {
		result.Gas = afterSwapGas
	}
	return result, nil
}

func (h *Hook) fee(gross *big.Int) (*big.Int, error) {
	if gross == nil || gross.Sign() < 0 || gross.BitLen() > 127 {
		return nil, uniswapv4.ErrInvalidAmountOut
	}
	if !h.Registered {
		return new(big.Int), nil
	}
	var amount, fee, tax, rate, denominator uint256.Int
	amount.SetFromBig(gross)
	denominator.SetUint64(basisPoints)
	// A positive int128 output times a rate bounded to 2000 bps fits uint256.
	fee.Mul(&amount, rate.SetUint64(uint64(h.HookFeeBps))).Div(&fee, &denominator)
	tax.Mul(&amount, rate.SetUint64(uint64(h.CreatorTaxBps))).Div(&tax, &denominator)
	return fee.Add(&fee, &tax).ToBig(), nil
}

func (h *Hook) CloneState() uniswapv4.Hook {
	clone := *h
	return &clone
}
