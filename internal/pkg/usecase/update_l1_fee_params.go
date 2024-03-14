package usecase

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/l2fee"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/l2fee/reader"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type UpdateL1FeeParams struct {
	handleFunc func(ctx context.Context) error
}

func NewUpdateL1FeeParams(
	chainId valueobject.ChainID,
	ethrpcClient *ethrpc.Client,
	oracleAddress string,
	paramsRepo *l2fee.RedisL1FeeRepository,
) *UpdateL1FeeParams {
	switch chainId {
	case valueobject.ChainIDScroll:
		paramsReader := reader.NewScrollFeeReader(ethrpcClient, oracleAddress)

		return &UpdateL1FeeParams{func(ctx context.Context) error {
			params, err := paramsReader.Read(ctx)
			if err != nil {
				return err
			}
			err = paramsRepo.UpdateL1FeeParams(ctx, params)
			return err
		}}
	}

	return nil
}

func (u *UpdateL1FeeParams) Handle(ctx context.Context) error {
	return u.handleFunc(ctx)
}
