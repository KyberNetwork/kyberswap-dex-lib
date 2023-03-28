package optimism

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
)

type FeeReader struct {
	contractAddress string
	abi             abi.ABI
	scanService     *service.ScanService
}

func NewOptimismFeeReader(
	scanService *service.ScanService,
	contractAddress string,
) *FeeReader {
	return &FeeReader{
		contractAddress: contractAddress,
		abi:             abis.OVMGasPriceOracle,
		scanService:     scanService,
	}
}

func (r *FeeReader) Read(ctx context.Context) (*entity.L2Fee, error) {
	var decimals, l1BaseFee, overhead, scalar *big.Int

	calls := []*repository.TryCallParams{
		{
			ABI:    r.abi,
			Target: r.contractAddress,
			Method: GasPriceOracleMethodDecimals,
			Params: nil,
			Output: &decimals,
		},
		{
			ABI:    r.abi,
			Target: r.contractAddress,
			Method: GasPriceOracleMethodL1BaseFee,
			Params: nil,
			Output: &l1BaseFee,
		},
		{
			ABI:    r.abi,
			Target: r.contractAddress,
			Method: GasPriceOracleMethodOverhead,
			Params: nil,
			Output: &overhead,
		},
		{
			ABI:    r.abi,
			Target: r.contractAddress,
			Method: GasPriceOracleMethodScalar,
			Params: nil,
			Output: &scalar,
		},
	}

	if err := r.scanService.TryAggregate(ctx, false, calls); err != nil {
		return nil, err
	}

	return &entity.L2Fee{
		Decimals:  decimals,
		L1BaseFee: l1BaseFee,
		Overhead:  overhead,
		Scalar:    scalar,
	}, nil
}
