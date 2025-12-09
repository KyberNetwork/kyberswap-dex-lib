package alpha

import (
	"context"
	"errors"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = cl.RegisterHooksFactory(func(param *cl.HookParam) cl.Hook {
	hook := &Hook{Hook: cl.NewBaseHook(valueobject.ExchangePancakeInfinityCLAlpha, param)}
	if len(param.HookExtra) > 0 {
		_ = json.Unmarshal(param.HookExtra, &hook.Extra)
	}
	return hook
},
	common.HexToAddress("0x9a9B5331ce8d74b2B721291D57DE696E878353fd"),
	common.HexToAddress("0x72e09eBd9b24F47730b651889a4eD984CBa53d90"),
)

type Hook struct {
	cl.Hook
	Extra
}

type Extra struct {
	StartTime int64 `json:"s"`
}

func (h *Hook) Track(ctx context.Context, param *cl.HookParam) ([]byte, error) {
	if len(param.HookExtra) > 0 {
		return param.HookExtra, nil
	}

	var extra Extra
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    Abi,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "poolStartedTimestamp",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&extra.StartTime}).TryBlockAndAggregate(); err != nil {
		return nil, err
	}

	return json.Marshal(extra)
}

var ErrPoolNotStarted = errors.New("pool not started")

func (h *Hook) BeforeSwap(_ *cl.BeforeSwapParams) (*cl.BeforeSwapResult, error) {
	if time.Now().Before(time.Unix(h.StartTime, 0)) {
		return nil, ErrPoolNotStarted
	}
	return &cl.BeforeSwapResult{
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
	}, nil
}
