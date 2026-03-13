package doppler

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook struct { // scheduled
	uniswapv4.Hook `json:"-"`
	StartingTime   int64
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	return &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Doppler}
}, NoopHookAddresses...)

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	var hook Hook
	if param.HookExtra != "" {
		_ = json.Unmarshal([]byte(param.HookExtra), &hook)
	}
	hook.Hook = &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Doppler}
	return &hook
}, ScheduledHookAddresses...)

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (string, error) {
	if len(param.HookExtra) > 0 {
		return param.HookExtra, nil
	}

	if _, err := param.RpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    hookABI,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "startingTimeOf",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&h.StartingTime}).Call(); err != nil {
		return "", err
	}

	extraBytes, _ := json.Marshal(h)
	return string(extraBytes), nil
}

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.StartingTime == 0 || time.Now().Unix() < h.StartingTime {
		return nil, ErrCannotSwapBeforeStartingTime
	}
	return h.Hook.BeforeSwap(params)
}
