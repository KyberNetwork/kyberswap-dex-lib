package bunniv2

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook struct {
	*uniswapv4.BaseHook
	hubCaller    *BunniV2HubContractCaller
	hubCallerErr error
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		BaseHook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4BunniV2},
	}
	hook.hubCaller, hook.hubCallerErr = NewBunniV2HubContractCaller(HubAddress, param.RpcClient.GetETHClient())
	return hook
}, HookAddresses...)

func (h *Hook) GetReserves(ctx context.Context, param *uniswapv4.HookParam) (entity.PoolReserves, error) {
	if err := h.hubCallerErr; err != nil {
		return nil, err
	}

	poolState, err := h.hubCaller.PoolState(&bind.CallOpts{Context: ctx}, common.HexToHash(param.Pool.Address))
	if err != nil {
		return nil, err
	}

	return entity.PoolReserves{
		poolState.Reserve0.Add(poolState.Reserve0, poolState.RawBalance0).String(),
		poolState.Reserve1.Add(poolState.Reserve1, poolState.RawBalance1).String(),
	}, nil
}
