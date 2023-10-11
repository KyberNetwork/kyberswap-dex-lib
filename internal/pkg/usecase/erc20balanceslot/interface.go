package erc20balanceslot

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

type ProbeStrategyExtraParams interface {
	ProbeStrategyExtraParams()
}

type ProbeStrategy interface {
	Name() string
	ProbeBalanceSlot(token common.Address, extraParams ProbeStrategyExtraParams) (*entity.ERC20BalanceSlot, error)
}
