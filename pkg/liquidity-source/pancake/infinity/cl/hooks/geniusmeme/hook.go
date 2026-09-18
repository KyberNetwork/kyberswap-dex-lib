package geniusmeme

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Hook ports Genius.fun's GeniusMemeHook: it only registers afterSwap (fee is
// 0 on the pool's LP side) and takes hookFeeBps() on the unspecified leg of
// every swap -- see https://genius.fun/docs/integrate "PancakeSwap Infinity".
var _ = cl.RegisterHooksFactory(func(param *cl.HookParam) cl.Hook {
	hook := &Hook{Hook: cl.NewBaseHook(valueobject.ExchangePancakeInfinityCLGeniusMeme, param)}
	if len(param.HookExtra) > 0 {
		_ = json.Unmarshal(param.HookExtra, &hook.Extra)
	}
	return hook
},
	common.HexToAddress("0xFf17F41c5Efd6CCe944Af0912F300097D62df5c9"),
)

type Hook struct {
	cl.Hook
	Extra
}

type Extra struct {
	FeeBps *big.Int `json:"f,omitempty"`
}

// Track reads hookFeeBps() live rather than hardcoding the documented 2%,
// since the docs explicitly warn it's owner-adjustable per pool.
func (h *Hook) Track(ctx context.Context, param *cl.HookParam) ([]byte, error) {
	if len(param.HookExtra) > 0 {
		return param.HookExtra, nil
	}

	var extra Extra
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    Abi,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "hookFeeBps",
	}, []any{&extra.FeeBps}).Aggregate(); err != nil {
		return nil, err
	}

	return json.Marshal(extra)
}

// AfterSwap takes the fee on the unspecified leg: CalcOut (exact-in) means
// the unspecified leg is the output, CalcIn (exact-out) means it's the input.
func (h *Hook) AfterSwap(params *cl.AfterSwapParams) (*cl.AfterSwapResult, error) {
	unspecifiedAmount := params.AmountIn
	if params.CalcOut {
		unspecifiedAmount = params.AmountOut
	}

	return &cl.AfterSwapResult{
		HookFee: bignumber.MulDivDown(new(big.Int), unspecifiedAmount, h.FeeBps, bignumber.BasisPoint),
	}, nil
}
