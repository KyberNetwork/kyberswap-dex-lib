package usecase

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/l2fee"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/l2fee/arbitrum"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/l2fee/optimism"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/l2fee/scroll"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type L1FeeEstimator struct {
	chainID    valueobject.ChainID
	paramsRepo *l2fee.RedisL1FeeRepository
}

func NewL1FeeEstimator(paramsRepo *l2fee.RedisL1FeeRepository, chainID valueobject.ChainID) *L1FeeEstimator {
	return &L1FeeEstimator{
		chainID:    chainID,
		paramsRepo: paramsRepo,
	}
}

func (r *L1FeeEstimator) EstimateL1Fees(ctx context.Context) (*big.Int, *big.Int, error) {
	l1FeeOverhead := big.NewInt(0)
	l1FeePerPool := big.NewInt(0)

	switch r.chainID {
	case valueobject.ChainIDArbitrumOne:
		var params entity.ArbitrumL1FeeParams
		if err := r.paramsRepo.GetL1FeeParams(ctx, &params); err != nil {
			return nil, nil, err
		}
		l1FeeOverhead, l1FeePerPool = arbitrum.EstimateL1Fees(&params)
	case valueobject.ChainIDOptimism, valueobject.ChainIDBase:
		var params entity.OptimismL1FeeParams
		if err := r.paramsRepo.GetL1FeeParams(ctx, &params); err != nil {
			return nil, nil, err
		}
		l1FeeOverhead, l1FeePerPool = optimism.EstimateFjordL1Fees(&params)
	case valueobject.ChainIDBlast:
		var params entity.OptimismL1FeeParams
		if err := r.paramsRepo.GetL1FeeParams(ctx, &params); err != nil {
			return nil, nil, err
		}
		l1FeeOverhead, l1FeePerPool = optimism.EstimateEcotoneL1Fees(&params)
	case valueobject.ChainIDScroll:
		var params entity.ScrollL1FeeParams
		if err := r.paramsRepo.GetL1FeeParams(ctx, &params); err != nil {
			return nil, nil, err
		}
		l1FeeOverhead, l1FeePerPool = scroll.EstimateL1Fees(&params)
	}
	return l1FeeOverhead, l1FeePerPool, nil
}
