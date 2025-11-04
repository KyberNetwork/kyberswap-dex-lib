package renzo

import (
	"context"
	"errors"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook struct {
	uniswapv4.Hook
	hook             string
	rate             *big.Int
	poolSqrtPriceX96 *big.Int
	minFeeBps        *big.Int
	maxFeeBps        *big.Int
}

type RenzoExtra struct {
	RateProviderAddress common.Address `json:"rP"`
	Rate                uint64         `json:"r"`
	MinFeeBps           uint64         `json:"minF"`
	MaxFeeBps           uint64         `json:"maxF"`
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Renzo},
		hook: param.HookAddress.Hex(),
	}

	if param.HookExtra != "" {
		var extra RenzoExtra
		if err := json.Unmarshal([]byte(param.HookExtra), &extra); err == nil {
			hook.rate = big.NewInt(int64(extra.Rate))
			hook.minFeeBps = big.NewInt(int64(extra.MinFeeBps))
			hook.maxFeeBps = big.NewInt(int64(extra.MaxFeeBps))
		}
	}

	if param.Pool != nil && param.Pool.Extra != "" {
		var extra uniswapv4.ExtraU256
		if err := json.Unmarshal([]byte(param.Pool.Extra), &extra); err == nil {
			if extra.ExtraTickU256 != nil && extra.SqrtPriceX96 != nil {
				hook.poolSqrtPriceX96 = extra.SqrtPriceX96.ToBig()
			}
		}
	}
	return hook
}, HookAddresses...)

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (string, error) {
	var extra RenzoExtra
	if param.HookExtra != "" {
		if err := json.Unmarshal([]byte(param.HookExtra), &extra); err != nil {
			return "", err
		}
	}

	if extra.RateProviderAddress == (common.Address{}) {
		if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).AddCall(&ethrpc.Call{
			ABI:    renzoHookABI,
			Target: h.hook,
			Method: "rateProvider",
		}, []any{&extra.RateProviderAddress}).Call(); err != nil {
			return "", err
		}
	}

	var rate, minFeeBps, maxFeeBps *big.Int
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).AddCall(&ethrpc.Call{
		ABI:    rateProviderABI,
		Target: extra.RateProviderAddress.Hex(),
		Method: "getRate",
	}, []any{&rate}).AddCall(&ethrpc.Call{
		ABI:    renzoHookABI,
		Target: h.hook,
		Method: "minFeeBps",
	}, []any{&minFeeBps}).AddCall(&ethrpc.Call{
		ABI:    renzoHookABI,
		Target: h.hook,
		Method: "maxFeeBps",
	}, []any{&maxFeeBps}).Aggregate(); err != nil {
		return "", err
	}

	extra.Rate = rate.Uint64()
	extra.MinFeeBps = minFeeBps.Uint64()
	extra.MaxFeeBps = maxFeeBps.Uint64()
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return "", err
	}

	return string(extraBytes), nil
}

func (h *Hook) BeforeSwap(swapHookParams *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.poolSqrtPriceX96 == nil || h.rate == nil {
		return nil, errors.New("sqrtPriceX96 or rate is not set")
	}
	referenceSqrtPriceX96 := exchangeRateToSqrtPriceX96(h.rate)
	var fee *big.Int
	if swapHookParams.ZeroForOne || h.poolSqrtPriceX96.Cmp(referenceSqrtPriceX96) < 0 {
		fee = h.minFeeBps
	} else {
		absPercentageDifferenceWad := absPercentageDifferenceWad(h.poolSqrtPriceX96, referenceSqrtPriceX96)
		fee = absPercentageDifferenceWad.Div(absPercentageDifferenceWad, B1e12)
		if fee.Cmp(h.minFeeBps) < 0 {
			fee = h.minFeeBps
		} else if fee.Cmp(h.maxFeeBps) > 0 {
			fee = h.maxFeeBps
		}
	}
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
		SwapFee:          uniswapv4.FeeAmount(fee.Uint64()),
	}, nil
}

func exchangeRateToSqrtPriceX96(rate *big.Int) *big.Int {
	num, den := new(big.Int), new(big.Int)
	num.Sqrt(WAD).Mul(num, q96).Div(num, den.Sqrt(rate))
	return num
}

func absPercentageDifferenceWad(sqrtPriceX96, denominatorX96 *big.Int) *big.Int {
	percentageDiffWad, divX96 := new(big.Int), new(big.Int)
	divX96.Mul(sqrtPriceX96, q96).Div(divX96, denominatorX96)
	percentageDiffWad.Mul(divX96, divX96).Mul(percentageDiffWad, WAD).Div(percentageDiffWad, q192).Sub(percentageDiffWad, WAD).Abs(percentageDiffWad)
	return percentageDiffWad
}
