package bunniv2

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	LegacyHookAddresses = []common.Address{
		common.HexToAddress("0x0010d0d5db05933fa0d9f7038d365e1541a41888"),
		common.HexToAddress("0x0000fe59823933ac763611a69c88f91d45f81888"),
	}

	LegacyHubAddress = common.HexToAddress("0x000000dceb71f3107909b1b748424349bfde5493")
)

type LegacyHook struct {
	uniswapv4.Hook
}

var _ = uniswapv4.RegisterHooksFactory(NewLegacyHook, LegacyHookAddresses...)

func NewLegacyHook(param *uniswapv4.HookParam) uniswapv4.Hook {
	return &LegacyHook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4BunniV2},
	}
}

func (h *LegacyHook) GetReserves(ctx context.Context, param *uniswapv4.HookParam) (entity.PoolReserves, error) {
	req := param.RpcClient.NewRequest().SetContext(ctx)

	var poolState LegacyPoolStateRPC
	req.AddCall(&ethrpc.Call{
		ABI:    legacyBunniHubABI,
		Target: LegacyHubAddress.Hex(),
		Method: "poolState",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&poolState})

	if _, err := req.Call(); err != nil {
		return nil, err
	}

	return entity.PoolReserves{
		poolState.Data.Reserve0.Add(poolState.Data.Reserve0, poolState.Data.RawBalance0).String(),
		poolState.Data.Reserve1.Add(poolState.Data.Reserve1, poolState.Data.RawBalance1).String(),
	}, nil
}
