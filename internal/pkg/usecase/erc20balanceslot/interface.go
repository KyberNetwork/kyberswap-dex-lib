package erc20balanceslot

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/types"
	dexentity "github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/ethereum/go-ethereum/common"
)

type ProbeStrategyExtraParams interface {
	ProbeStrategyExtraParams()
}

type ProbeStrategy interface {
	Name(extraParams ProbeStrategyExtraParams) string
	ProbeBalanceSlot(ctx context.Context, token common.Address, extraParams ProbeStrategyExtraParams) (*types.ERC20BalanceSlot, error)
}

type ICache interface {
	PreloadMany(ctx context.Context, tokens []common.Address) error
	PreloadFromEmbedded(ctx context.Context) error
	Get(ctx context.Context, token common.Address, pool *dexentity.Pool) (*types.ERC20BalanceSlot, error)
}
