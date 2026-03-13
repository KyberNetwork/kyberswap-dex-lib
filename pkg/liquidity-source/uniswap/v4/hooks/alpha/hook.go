package alpha

import (
	"context"
	"errors"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	var hook Hook
	if param.HookExtra != "" {
		_ = json.Unmarshal([]byte(param.HookExtra), &hook)
	}
	hook.Hook = &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Alpha}
	return &hook
},
	common.HexToAddress("0xb0Ba5d56364569496e0aA5158C3242420eaDE880"), // base
	common.HexToAddress("0xB0BcB37dD65712c3Afa101D54389c8279659A880"), // bsc
)

type Hook struct {
	uniswapv4.Hook `json:"-"`
	StartTime      int64 `json:"s"`
}

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (string, error) {
	if len(param.HookExtra) > 0 {
		return param.HookExtra, nil
	}

	if _, err := param.RpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    Abi,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "poolStartedTimestamp",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&h.StartTime}).TryBlockAndAggregate(); err != nil {
		return "", err
	}

	extraBytes, _ := json.Marshal(h)
	return string(extraBytes), nil
}

var ErrPoolNotStarted = errors.New("pool not started")

func (h *Hook) BeforeSwap(_ *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if time.Now().Before(time.Unix(h.StartTime, 0)) {
		return nil, ErrPoolNotStarted
	}
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
	}, nil
}
