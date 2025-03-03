package scroll

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
	var l1BaseFee, commitScalar, blobScalar, l1BlobBaseFee *big.Int

	calls := r.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    abis.ScrolL1GasPriceOracle,
		Target: l1GasPriceOracleAddress,
		Method: methodL1BaseFee,
	}, []interface{}{&l1BaseFee})
	calls.AddCall(&ethrpc.Call{
		ABI:    abis.ScrolL1GasPriceOracle,
		Target: l1GasPriceOracleAddress,
		Method: methodL1CommitScalar,
	}, []interface{}{&commitScalar})
	calls.AddCall(&ethrpc.Call{
		ABI:    abis.ScrolL1GasPriceOracle,
		Target: l1GasPriceOracleAddress,
		Method: methodL1BlobScalar,
	}, []interface{}{&blobScalar})
	calls.AddCall(&ethrpc.Call{
		ABI:    abis.ScrolL1GasPriceOracle,
		Target: l1GasPriceOracleAddress,
		Method: methodL1BlobBaseFee,
	}, []interface{}{&l1BlobBaseFee})
	if _, err := calls.Aggregate(); err != nil {
		return nil, fmt.Errorf("failed to aggregate call to get Scroll fee params %v", err)
	}

	return &entity.ScrollL1FeeParams{
		L1BaseFee:      l1BaseFee,
		L1CommitScalar: commitScalar,
		L1BlobScalar:   blobScalar,
		L1BlobBaseFee:  l1BlobBaseFee,
	}, nil
}
