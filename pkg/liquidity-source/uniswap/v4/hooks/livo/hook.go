package livo

import (
	"context"
	"errors"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	ErrNoSwapsBeforeGraduation = errors.New("no swaps before graduation")
	ErrPoolTokensNotAvailable  = errors.New("pool tokens not available")
)

type Hook struct {
	uniswapv4.Hook      `json:"-"`
	Graduated           bool   `json:"g,omitempty"`
	BuyTaxBps           uint64 `json:"b,omitempty"`
	SellTaxBps          uint64 `json:"s,omitempty"`
	TaxDurationSeconds  uint64 `json:"d,omitempty"`
	GraduationTimestamp uint64 `json:"t,omitempty"`
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Livo},
	}
	_ = param.HookExtra.Unmarshal(&hook)
	return hook
}, HookAddresses...)

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if len(param.HookExtra) > 0 {
		return json.RawMessage(param.HookExtra), nil
	} else if param.Pool == nil || len(param.Pool.Tokens) < 2 {
		return nil, ErrPoolTokensNotAvailable
	}

	// tokenAddress is currency1 (the token being swapped)
	tokenAddress := param.Pool.Tokens[1].Address

	var (
		graduated bool
		taxConfig struct {
			R struct {
				BuyTaxBps           uint16
				SellTaxBps          uint16
				TaxDurationSeconds  uint64
				GraduationTimestamp uint64
			}
		}
	)

	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).AddCall(&ethrpc.Call{
		ABI:    ILivoTokenABI,
		Target: tokenAddress,
		Method: "graduated",
	}, []any{&graduated}).AddCall(&ethrpc.Call{
		ABI:    ILivoTokenABI,
		Target: tokenAddress,
		Method: "getTaxConfig",
	}, []any{&taxConfig}).Aggregate(); err != nil {
		return nil, err
	}

	h.Graduated = graduated
	h.BuyTaxBps = uint64(taxConfig.R.BuyTaxBps)
	h.SellTaxBps = uint64(taxConfig.R.SellTaxBps)
	h.TaxDurationSeconds = taxConfig.R.TaxDurationSeconds
	h.GraduationTimestamp = taxConfig.R.GraduationTimestamp

	return json.Marshal(h)
}

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	// Check if token has graduated
	if !h.Graduated {
		return nil, ErrNoSwapsBeforeGraduation
	} else if !params.ZeroForOne {
		// For buys (zeroForOne=true): charge LP fee + buy tax on ETH input
		// For sells (zeroForOne=false): no action in beforeSwap
		return &uniswapv4.BeforeSwapResult{
			DeltaSpecified:   new(uint256.Int),
			DeltaUnspecified: new(uint256.Int),
		}, nil
	}

	// BUY: charge LP fee + buy tax
	amountSpecified := params.AmountSpecified

	var lpFee, taxAmount uint256.Int
	big256.MulDivDown(&lpFee, amountSpecified, uint256.NewInt(LpFeeBps), big256.UBasisPoint)

	// Check if tax period has expired
	currentTime := uint64(time.Now().Unix())
	if h.GraduationTimestamp > 0 && currentTime <= h.GraduationTimestamp+h.TaxDurationSeconds && h.BuyTaxBps > 0 {
		big256.MulDivDown(&taxAmount, amountSpecified, uint256.NewInt(h.BuyTaxBps), big256.UBasisPoint)
	}

	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   lpFee.Add(&lpFee, &taxAmount),
		DeltaUnspecified: new(uint256.Int),
	}, nil
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	// Check if token has graduated
	if !h.Graduated {
		return nil, ErrNoSwapsBeforeGraduation
	} else if params.ZeroForOne { // For buys: fees already charged in beforeSwap, no additional fee
		return &uniswapv4.AfterSwapResult{
			HookFee: new(uint256.Int),
		}, nil
	}

	// SELL: charge LP fee + sell tax on ETH output
	amountOut := params.AmountOut

	var lpFee, taxAmount uint256.Int
	big256.MulDivDown(&lpFee, amountOut, uint256.NewInt(LpFeeBps), big256.UBasisPoint)

	// Check if tax period has expired
	currentTime := uint64(time.Now().Unix())
	if h.GraduationTimestamp > 0 && currentTime <= h.GraduationTimestamp+h.TaxDurationSeconds && h.SellTaxBps > 0 {
		big256.MulDivDown(&taxAmount, amountOut, uint256.NewInt(h.SellTaxBps), big256.UBasisPoint)
	}

	return &uniswapv4.AfterSwapResult{
		HookFee: lpFee.Add(&lpFee, &taxAmount),
	}, nil
}
