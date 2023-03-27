package arbitrum

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
)

type FeeReader struct {
	contractAddress string
	abi             abi.ABI
	scanService     *service.ScanService
}

func NewArbitrumFeeReader(
	scanService *service.ScanService,
	contractAddress string,
) *FeeReader {
	return &FeeReader{
		contractAddress: contractAddress,
		abi:             abis.ArbGasInfo,
		scanService:     scanService,
	}
}

func (r *FeeReader) Read(ctx context.Context) (*entity.L2Fee, error) {
	var l1BaseFee *big.Int

	calls := []*repository.TryCallParams{
		{
			ABI:    r.abi,
			Target: r.contractAddress,
			Method: GasInfoMethodGetL1BaseFeeEstimate,
			Params: nil,
			Output: &l1BaseFee,
		},
	}

	if err := r.scanService.TryAggregate(ctx, false, calls); err != nil {
		return nil, err
	}

	return &entity.L2Fee{
		L1BaseFee: l1BaseFee,
	}, nil
}
