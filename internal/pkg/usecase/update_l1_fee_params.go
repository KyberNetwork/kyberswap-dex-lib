package usecase

import (
	"context"

	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/router-service/internal/pkg/repository/l2fee"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/l2fee/arbitrum"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/l2fee/optimism"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/l2fee/scroll"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type UpdateL1FeeParams struct {
	handleFunc func(ctx context.Context) error
}

type FeeReader interface {
	Read(ctx context.Context) (any, error)
}

func NewUpdateL1FeeParams(
	chainId valueobject.ChainID,
	ethrpcClient *ethrpc.Client,
	paramsRepo *l2fee.RedisL1FeeRepository,
) *UpdateL1FeeParams {
	var reader FeeReader
	switch chainId {
	case valueobject.ChainIDOptimism, valueobject.ChainIDBase, valueobject.ChainIDBlast:
		reader = optimism.NewFeeReader(ethrpcClient)
	case valueobject.ChainIDArbitrumOne:
		reader = arbitrum.NewFeeReader(ethrpcClient)
	case valueobject.ChainIDScroll:
		reader = scroll.NewFeeReader(ethrpcClient)
	default:
		return nil
	}

	return &UpdateL1FeeParams{func(ctx context.Context) error {
		params, err := reader.Read(ctx)
		if err != nil {
			return err
		}
		err = paramsRepo.UpdateL1FeeParams(ctx, params)
		return err
	}}
}

func (u *UpdateL1FeeParams) Handle(ctx context.Context) error {
	return u.handleFunc(ctx)
}
