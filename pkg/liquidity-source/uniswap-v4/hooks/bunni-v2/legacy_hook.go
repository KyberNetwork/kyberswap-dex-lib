package bunniv2

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

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
	hubCaller    *BunniV2HubContractCaller
	hubCallerErr error
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &LegacyHook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4BunniV2},
	}
	if param.RpcClient == nil {
		hook.hubCallerErr = errors.New("nil rpc client")
	} else {
		hook.hubCaller, hook.hubCallerErr = NewBunniV2HubContractCaller(LegacyHubAddress, param.RpcClient.GetETHClient())
	}
	return hook
}, LegacyHookAddresses...)

func (h *LegacyHook) GetReserves(ctx context.Context, param *uniswapv4.HookParam) (entity.PoolReserves, error) {
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
