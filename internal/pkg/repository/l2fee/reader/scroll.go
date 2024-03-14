package reader

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

const (
	methodL1BaseFee = "l1BaseFee"
	methodScalar    = "scalar"
	methodOverhead  = "overhead"
)

type scrollFeeReader struct {
	oracleAddress string
	ethrpcClient  *ethrpc.Client
}

func NewScrollFeeReader(
	ethrpcClient *ethrpc.Client,
	oracleAddress string,
) *scrollFeeReader {
	return &scrollFeeReader{
		oracleAddress: oracleAddress,
		ethrpcClient:  ethrpcClient,
	}
}

func (r *scrollFeeReader) Read(ctx context.Context) (*entity.ScrollL1FeeParams, error) {
	var l1BaseFee, scalar, overhead *big.Int

	calls := r.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    abis.ScrolL1GasPriceOracle,
		Target: r.oracleAddress,
		Method: methodL1BaseFee,
		Params: nil,
	}, []interface{}{&l1BaseFee})
	calls.AddCall(&ethrpc.Call{
		ABI:    abis.ScrolL1GasPriceOracle,
		Target: r.oracleAddress,
		Method: methodScalar,
		Params: nil,
	}, []interface{}{&scalar})
	calls.AddCall(&ethrpc.Call{
		ABI:    abis.ScrolL1GasPriceOracle,
		Target: r.oracleAddress,
		Method: methodOverhead,
		Params: nil,
	}, []interface{}{&overhead})

	if _, err := calls.TryAggregate(); err != nil {
		logger.Errorf(ctx, "failed to aggregate call to get Scroll fee param %v", err)
		return nil, err
	}

	return &entity.ScrollL1FeeParams{
		L1BaseFee: l1BaseFee,
		Overhead:  overhead,
		Scalar:    scalar,
	}, nil
}
