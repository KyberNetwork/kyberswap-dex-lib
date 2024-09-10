package erc20balanceslot

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/types"
	"github.com/ethereum/go-ethereum/common"
)

type ProbeStrategyExtraParams interface {
	ProbeStrategyExtraParams()
}

type ProbeStrategy interface {
	Name(extraParams ProbeStrategyExtraParams) string
	ProbeBalanceSlot(ctx context.Context, token common.Address, extraParams ProbeStrategyExtraParams) (*types.ERC20BalanceSlot, error)
}
