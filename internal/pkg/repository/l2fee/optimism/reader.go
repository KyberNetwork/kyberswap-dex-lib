package optimism

import (
	"context"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

type feeReader struct {
	ethrpcClient *ethrpc.Client
}

func NewFeeReader(
	ethrpcClient *ethrpc.Client,
) *feeReader {
	return &feeReader{
		ethrpcClient: ethrpcClient,
	}
}

func (r *feeReader) Read(ctx context.Context) (any, error) {
	var (
		l1BaseFee           *big.Int
		l1BaseFeeScalar     uint32
		l1BlobBaseFee       *big.Int
		l1BlobBaseFeeScalar uint32
	)

	calls := r.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    abis.OptimismGasPriceOracle,
		Target: gasPriceOracleAddress,
		Method: methodGetL1BaseFee,
	}, []interface{}{&l1BaseFee})
	calls.AddCall(&ethrpc.Call{
		ABI:    abis.OptimismGasPriceOracle,
		Target: gasPriceOracleAddress,
		Method: methodGetL1BaseFeeScalar,
	}, []interface{}{&l1BaseFeeScalar})
	calls.AddCall(&ethrpc.Call{
		ABI:    abis.OptimismGasPriceOracle,
		Target: gasPriceOracleAddress,
		Method: methodGetL1BlobBaseFee,
	}, []interface{}{&l1BlobBaseFee})
	calls.AddCall(&ethrpc.Call{
		ABI:    abis.OptimismGasPriceOracle,
		Target: gasPriceOracleAddress,
		Method: methodGetL1BlobBaseFeeScalar,
	}, []interface{}{&l1BlobBaseFeeScalar})
	if _, err := calls.Aggregate(); err != nil {
		return nil, fmt.Errorf("failed to aggregate call to get fee param %v", err)
	}

	return &entity.OptimismL1FeeParams{
		L1BaseFee:           l1BaseFee,
		L1BaseFeeScalar:     big.NewInt(int64(l1BaseFeeScalar)),
		L1BlobBaseFee:       l1BlobBaseFee,
		L1BlobBaseFeeScalar: big.NewInt(int64(l1BlobBaseFeeScalar)),
	}, nil
}
