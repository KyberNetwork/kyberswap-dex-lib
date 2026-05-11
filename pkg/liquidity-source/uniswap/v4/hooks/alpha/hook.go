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
	hook := Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Alpha},
	}
	_ = param.HookExtra.Unmarshal(&hook)
	return &hook
},
	common.HexToAddress("0xb0Ba5d56364569496e0aA5158C3242420eaDE880"), // base
	common.HexToAddress("0xB0BcB37dD65712c3Afa101D54389c8279659A880"), // bsc
	common.HexToAddress("0xB0b24B89dB0dafbE43C5b40226b63A179f592880"), // base uniswap v4
	common.HexToAddress("0xB0Be14859E2cA735B22E58C52A6F3413454E2880"), // arb uniswap v4
)

type Hook struct {
	uniswapv4.Hook `json:"-"`
	StartTime      int64 `json:"s"`
}

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if len(param.HookExtra) > 0 {
		return json.RawMessage(param.HookExtra), nil
	}

	if _, err := param.RpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    Abi,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "poolStartedTimestamp",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&h.StartTime}).TryBlockAndAggregate(); err != nil {
		return nil, err
	}

	return json.Marshal(h)
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
