package alphix

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// ProHook implements the AlphixPro hook — an asymmetric dynamic fee hook
// for partner token pools. Multi-pool capable: a single hook address can
// serve many pools, each with its own independent buy/sell fees and quote
// orientation, tracked per-pool via Track().
//
// The applied fee depends on swap direction relative to the quote token.
// Unlike LvrFeeHook, there is no hook fee (no protocol fee capture on output).
// BeforeSwap returns either buyFee or sellFee; AfterSwap is a no-op.
type ProHook struct {
	uniswapv4.Hook `json:"-"`
	BuyFee         uniswapv4.FeeAmount `json:"bf"`
	SellFee        uniswapv4.FeeAmount `json:"sf"`
	IsToken0Quote  bool                `json:"q"`
}

type ProExtra struct {
	BuyFee        uint32 `json:"bf"`
	SellFee       uint32 `json:"sf"`
	IsToken0Quote bool   `json:"q"`
}

type ProPoolConfig struct {
	BuyFee        uint32 `json:"bf"`
	SellFee       uint32 `json:"sf"`
	IsToken0Quote bool   `json:"q"`
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &ProHook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Alphix},
	}
	var extra ProExtra
	if err := param.HookExtra.Unmarshal(&extra); err == nil {
		hook.BuyFee = uniswapv4.FeeAmount(extra.BuyFee)
		hook.SellFee = uniswapv4.FeeAmount(extra.SellFee)
		hook.IsToken0Quote = extra.IsToken0Quote
	}
	return hook
}, ProHookAddresses...)

func (h *ProHook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	hookTarget := hexutil.Encode(param.HookAddress[:])
	poolId := common.HexToHash(param.Pool.Address)

	var proPoolConfig ProPoolConfig
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		AddCall(&ethrpc.Call{
			ABI:    proHookABI,
			Target: hookTarget,
			Method: "getPoolConfig",
			Params: []any{poolId},
		}, []any{&proPoolConfig}).
		Aggregate(); err != nil {
		return nil, err
	}
	return json.Marshal(ProExtra{
		BuyFee:        proPoolConfig.BuyFee,
		SellFee:       proPoolConfig.SellFee,
		IsToken0Quote: proPoolConfig.IsToken0Quote,
	})
}

func (h *ProHook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	// If isToken0Quote: zeroForOne means input=quote → buying project token → buyFee
	// If !isToken0Quote: zeroForOne means input=project → selling project token → sellFee
	fee := h.SellFee
	if h.IsToken0Quote == params.ZeroForOne {
		fee = h.BuyFee
	}
	return &uniswapv4.BeforeSwapResult{
		SwapFee:          fee,
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
	}, nil
}

func (h *ProHook) CloneState() uniswapv4.Hook {
	cloned := *h
	return &cloned
}
