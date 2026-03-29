package alphixlvrfee

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

// MaxHookFee is 1e6, matching the on-chain MAX_HOOK_FEE constant.
var MaxHookFee = bignumber.TenPowInt(6)

type Hook struct {
	uniswapv4.Hook `json:"-"`
	SwapFee        uniswapv4.FeeAmount `json:"f"`
	HookFee        int64               `json:"hf"`
}

type Extra struct {
	SwapFee int64 `json:"f"`
	HookFee int64 `json:"hf"`
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4AlphixLvrFee},
	}
	var extra Extra
	if err := param.HookExtra.Unmarshal(&extra); err == nil {
		hook.SwapFee = uniswapv4.FeeAmount(extra.SwapFee)
		hook.HookFee = extra.HookFee
	}
	return hook
}, HookAddresses...)

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	hookTarget := hexutil.Encode(param.HookAddress[:])
	poolId := common.HexToHash(param.Pool.Address)

	var swapFee, hookFee *big.Int
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		AddCall(&ethrpc.Call{
			ABI:    hookABI,
			Target: hookTarget,
			Method: "getFee",
			Params: []any{poolId},
		}, []any{&swapFee}).
		AddCall(&ethrpc.Call{
			ABI:    hookABI,
			Target: hookTarget,
			Method: "getHookFee",
			Params: []any{poolId},
		}, []any{&hookFee}).
		Aggregate(); err != nil {
		return nil, err
	}
	return json.Marshal(Extra{
		SwapFee: swapFee.Int64(),
		HookFee: hookFee.Int64(),
	})
}

func (h *Hook) BeforeSwap(_ *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	return &uniswapv4.BeforeSwapResult{
		SwapFee:          h.SwapFee,
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
	}, nil
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if h.HookFee == 0 {
		return &uniswapv4.AfterSwapResult{HookFee: bignumber.ZeroBI}, nil
	}
	return &uniswapv4.AfterSwapResult{
		HookFee: bignumber.MulDivDown(new(big.Int),
			lo.Ternary(params.CalcOut, params.AmountOut, params.AmountIn), big.NewInt(h.HookFee), MaxHookFee),
	}, nil
}

func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h
	return &cloned
}
