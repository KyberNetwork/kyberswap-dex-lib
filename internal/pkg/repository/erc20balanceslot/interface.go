package erc20balanceslot

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/types"
	"github.com/ethereum/go-ethereum/common"
)

type IRepository interface {
	Count(ctx context.Context) (int, error)
	Get(ctx context.Context, token common.Address) (*types.ERC20BalanceSlot, error)
	GetMany(ctx context.Context, tokens []common.Address) (map[common.Address]*types.ERC20BalanceSlot, error)
	GetAll(ctx context.Context) (map[common.Address]*types.ERC20BalanceSlot, error)
	Put(ctx context.Context, balanceSlot *types.ERC20BalanceSlot) error
	PutMany(ctx context.Context, balanceSlots []*types.ERC20BalanceSlot) error
}
