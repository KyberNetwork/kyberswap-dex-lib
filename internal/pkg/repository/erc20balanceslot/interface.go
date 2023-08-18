package erc20balanceslot

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/ethereum/go-ethereum/common"
)

type IRepository interface {
	GetPrefix() string
	Get(ctx context.Context, token common.Address) (*entity.ERC20BalanceSlot, error)
	GetAll(ctx context.Context) (map[common.Address]*entity.ERC20BalanceSlot, error)
	PutMany(ctx context.Context, balanceSlots []*entity.ERC20BalanceSlot) error
}
