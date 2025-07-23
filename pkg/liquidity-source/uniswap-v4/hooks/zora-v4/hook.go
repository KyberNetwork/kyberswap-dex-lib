package contentcoin

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook struct {
	*uniswapv4.BaseHook
	hook common.Address
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		BaseHook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4ZoraV4},
		hook:     param.HookAddress,
	}

	return hook
}, HookAddresses...)

func (h *Hook) GetReserves(ctx context.Context, param *uniswapv4.HookParam) (entity.PoolReserves, error) {
	return nil, nil
}

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (string, error) {
	return "", nil
}

func (h *Hook) BeforeSwap() (hookFeeAmt *big.Int, swapFee uniswapv4.FeeAmount) {
	// No beforeSwap logic for this hook
	return
}

func (h *Hook) AfterSwap() (hookFeeAmt *big.Int) {
	// The main logic is to convert remaining fees to payout currency
	// for market rewards. It doesn't modify the amountOut and only for
	// their dex internal purposes. So empty implementation here.
	return nil
}
