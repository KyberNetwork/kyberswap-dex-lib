package arbitrum

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
	var l1BaseFee *big.Int

	calls := r.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    abis.ArbGasInfo,
		Target: arbGasInfoAddress,
		Method: methodGetL1BaseFeeEstimate,
	}, []interface{}{&l1BaseFee})
	if _, err := calls.Aggregate(); err != nil {
		return nil, fmt.Errorf("failed to aggregate call to get Arbitrum fee param %v", err)
	}

	return &entity.ArbitrumL1FeeParams{
		L1BaseFee: l1BaseFee,
	}, nil
}
