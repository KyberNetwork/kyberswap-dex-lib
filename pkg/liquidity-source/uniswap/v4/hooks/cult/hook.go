package cult

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook struct {
	uniswapv4.Hook
	totalFeeBps *uint256.Int
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
		hook.totalFeeBps = uint256.NewInt(uint64(extra.TotalFeeBps))
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
	feeAmt := new(uint256.Int)
	if params.ZeroForOne && params.CalcOut {
		feeAmt.Mul(params.AmountSpecified, h.totalFeeBps).Div(feeAmt, big256.UBasisPoint)
	}

	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   feeAmt,
		DeltaUnspecified: new(uint256.Int),
	}, nil
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	feeAmt := new(uint256.Int)
	if !params.ZeroForOne {
		big256.MulDivDown(feeAmt, lo.Ternary(params.CalcOut, params.AmountOut, params.AmountIn),
			h.totalFeeBps, big256.UBasisPoint)
	}

	return &uniswapv4.AfterSwapResult{
		HookFee: feeAmt,
	}, nil
}

var HookData = common.FromHex("0000000000000000000000001111110f0f73c0b2ef09ec012eae758b3e03a902" +
	"0000000000000000000000001111110f0f73c0b2ef09ec012eae758b3e03a902")

func (h *Hook) GetHookData() []byte {
	return HookData
}
