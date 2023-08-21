package erc20balanceslot

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

type TODO struct {
}

func (*TODO) Get(_ context.Context, _ common.Address) (*entity.ERC20BalanceSlot, error) {
	panic("todo")
}
