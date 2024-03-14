package usecase

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/l2fee"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/l2fee/calculator"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type L1FeeCalculator struct {
	paramsRepo    *l2fee.RedisL1FeeRepository
	routerAddress common.Address
}

var (
	defaultTxNativeValue = big.NewInt(0)
)

func NewL1FeeCalculator(paramsRepo *l2fee.RedisL1FeeRepository, routerAddress common.Address) *L1FeeCalculator {
	return &L1FeeCalculator{
		paramsRepo:    paramsRepo,
		routerAddress: routerAddress,
	}
}

func (c *L1FeeCalculator) CalculateL1Fee(ctx context.Context, chainId valueobject.ChainID, encodedSwapData string) (*big.Int, error) {
	switch chainId {
	case valueobject.ChainIDScroll:
		var params entity.ScrollL1FeeParams
		err := c.paramsRepo.GetL1FeeParams(ctx, &params)
		if err != nil {
			return nil, err
		}
		tx, err := c.getUnsignedTx(encodedSwapData)
		if err != nil {
			return nil, err
		}
		return calculator.CalculateScrollL1Fee(params, tx)
	}

	return nil, nil
}

func (c *L1FeeCalculator) getUnsignedTx(encodedSwapData string) (*types.Transaction, error) {
	encodedData, err := hexutil.Decode(encodedSwapData)
	if err != nil {
		return nil, err
	}
	tx := types.NewTransaction(0, c.routerAddress, defaultTxNativeValue, 0, nil, encodedData)
	return tx, nil
}
