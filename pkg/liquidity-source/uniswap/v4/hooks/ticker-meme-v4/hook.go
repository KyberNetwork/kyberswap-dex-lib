// Package tickermemev4 implements the afterSwap fee of TickerMemeHook's classic
// Uniswap V4 pools. The V4 simulator already supplies tick pricing, LP fees,
// native currency wrapping, pool discovery and state updates.
package tickermemev4

import (
	"math/big"

	"github.com/holiman/uint256"

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
		// Decode atomically: decodeErr is not serialized, so a partial decode
		// must not retain a verified implementation hash and incomplete fees.
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
	if h.Registered && (h.HookFeeBps > maxHookFeeBps || uint32(h.HookFeeBps)+uint32(h.CreatorTaxBps) > maxTradeFeeBps) {
		return ErrInvalidTerms
	}
	return nil
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if err := h.validate(); err != nil {
		return nil, err
	}
	// CalcAmountIn is handled by ReverseExactInput, not the contract's exact-out
	// input surcharge. The router executes its result as an exact-input swap.
	if !params.CalcOut {
		return nil, uniswapv4.ErrInvalidAmountOut
	}
	if params.AmountOut == nil || params.AmountOut.Sign() <= 0 {
		return nil, uniswapv4.ErrInvalidAmountOut
	}
	fee, err := h.fee(params.AmountOut)
	if err != nil {
		return nil, err
	}
	gas := int64(0)
	if fee.Sign() > 0 {
		gas = afterSwapGas
	}
	return &uniswapv4.AfterSwapResult{HookFee: fee, Gas: gas}, nil
}

// fee mirrors floor(gross*hookFeeBps/10000) + floor(gross*creatorTaxBps/10000).
func (h *Hook) fee(gross *big.Int) (*big.Int, error) {
	if gross == nil || gross.Sign() < 0 || gross.BitLen() > 127 {
		return nil, uniswapv4.ErrInvalidAmountOut
	}
	if !h.Registered {
		return new(big.Int), nil
	}
	var amount, fee, tax, rate, denominator uint256.Int
	if amount.SetFromBig(gross) {
		return nil, uniswapv4.ErrInvalidAmountOut
	}
	denominator.SetUint64(basisPoints)
	// gross is a positive int128 delta and rates are bounded to 2000 bps, so
	// each product fits uint256. Keep the two Solidity divisions separate.
	fee.Mul(&amount, rate.SetUint64(uint64(h.HookFeeBps))).Div(&fee, &denominator)
	tax.Mul(&amount, rate.SetUint64(uint64(h.CreatorTaxBps))).Div(&tax, &denominator)
	return fee.Add(&fee, &tax).ToBig(), nil
}

func (h *Hook) CloneState() uniswapv4.Hook {
	clone := *h
	return &clone
}

var _ uniswapv4.Hook = (*Hook)(nil)
