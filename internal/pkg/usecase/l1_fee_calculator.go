package usecase

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/l2fee"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/l2fee/optimism"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/l2fee/scroll"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type L1FeeCalculator struct {
	paramsRepo    *l2fee.RedisL1FeeRepository
	chainID       valueobject.ChainID
	routerAddress common.Address
}

var (
	defaultTxNativeValue = big.NewInt(0)
)

func NewL1FeeCalculator(
	paramsRepo *l2fee.RedisL1FeeRepository,
	chainID valueobject.ChainID,
	routerAddress common.Address,
) *L1FeeCalculator {
	return &L1FeeCalculator{
		paramsRepo:    paramsRepo,
		chainID:       chainID,
		routerAddress: routerAddress,
	}
}

func (c *L1FeeCalculator) CalculateL1Fee(
	ctx context.Context,
	routeSummary valueobject.RouteSummary,
	encodedSwapData string,
) (*big.Int, error) {
	if !valueobject.IsL1FeeEstimateSupported(c.chainID) {
		return nil, nil
	}

	tx, err := c.getUnsignedTx(routeSummary, encodedSwapData)
	if err != nil {
		return nil, err
	}

	switch c.chainID {
	case valueobject.ChainIDScroll:
		var params entity.ScrollL1FeeParams
		if err = c.paramsRepo.GetL1FeeParams(ctx, &params); err != nil {
			return nil, err
		}
		return scroll.CalcCurieL1Fee(&params, tx)
	case valueobject.ChainIDOptimism, valueobject.ChainIDBase:
		var params entity.OptimismL1FeeParams
		if err = c.paramsRepo.GetL1FeeParams(ctx, &params); err != nil {
			return nil, err
		}
		return optimism.CalcFjordL1Fee(&params, tx)
	case valueobject.ChainIDBlast:
		var params entity.OptimismL1FeeParams
		if err = c.paramsRepo.GetL1FeeParams(ctx, &params); err != nil {
			return nil, err
		}
		return optimism.CalcEcotoneL1Fee(&params, tx)
	}

	return nil, nil
}

func (c *L1FeeCalculator) getUnsignedTx(routeSummary valueobject.RouteSummary, encodedSwapData string) (*types.Transaction, error) {
	encodedData, err := hexutil.Decode(encodedSwapData)
	if err != nil {
		return nil, err
	}
	gasPrice, _ := routeSummary.GasPrice.Int(nil)
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    0,
		GasPrice: gasPrice,
		Gas:      uint64(routeSummary.Gas),
		To:       &c.routerAddress,
		Value:    defaultTxNativeValue,
		Data:     encodedData,
	})
	return tx, nil
}
