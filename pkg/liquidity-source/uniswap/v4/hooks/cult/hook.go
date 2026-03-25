package cult

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook struct {
	uniswapv4.Hook
	totalFeeBps *big.Int
}

type Extra struct {
	TotalFeeBps int64 `json:"f"`
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Cult},
	}

	var extra Extra
	if err := param.HookExtra.Unmarshal(&extra); err == nil {
		hook.totalFeeBps = big.NewInt(extra.TotalFeeBps)
	}
	return hook
}, lo.Keys(FactoryByHook)...)

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	var extra Extra
	factory := FactoryByHook[param.HookAddress]
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).AddCall(&ethrpc.Call{
		ABI:    hookABI,
		Target: hexutil.Encode(factory[:]),
		Method: "totalFeeBps",
	}, []any{&extra.TotalFeeBps}).Call(); err != nil {
		return nil, err
	}
	return json.Marshal(extra)
}

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	feeAmt := bignumber.ZeroBI
	if params.ZeroForOne && params.ExactIn {
		feeAmt = new(big.Int)
		feeAmt.Mul(params.AmountSpecified, h.totalFeeBps).Div(feeAmt, bignumber.BasisPoint)
	}

	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   feeAmt,
		DeltaUnspecified: bignumber.ZeroBI,
	}, nil
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	feeAmt := bignumber.ZeroBI
	if !params.ZeroForOne {
		feeAmt = bignumber.MulDivDown(new(big.Int), lo.Ternary(params.ExactIn, params.AmountOut, params.AmountIn),
			h.totalFeeBps, bignumber.BasisPoint)
	}

	return &uniswapv4.AfterSwapResult{
		HookFee: feeAmt,
	}, nil
}

var HookData = common.FromHex("0000000000000000000000004f82e73edb06d29ff62c91ec8f5ff06571bdeb29" +
	"0000000000000000000000004f82e73edb06d29ff62c91ec8f5ff06571bdeb29")

func (h *Hook) GetHookData() []byte {
	return HookData
}
