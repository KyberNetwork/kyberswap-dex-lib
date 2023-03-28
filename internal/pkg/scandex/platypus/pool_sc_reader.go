package platypus

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/service"

	"github.com/ethereum/go-ethereum/common"
)

type PoolSCField string

const (
	PoolSCFieldC1             = "c1"
	PoolSCFieldHaircutRate    = "haircutRate"
	PoolSCFieldPriceOracle    = "priceOracle"
	PoolsCFieldRetentionRatio = "retentionRatio"
	PoolSCFieldSlippageParamK = "slippageParamK"
	PoolSCFieldSlippageParamN = "slippageParamN"
	PoolSCFieldTokenAddresses = "tokenAddresses"
	PoolSCFieldXThreshold     = "xThreshold"
	PoolSCFieldPaused         = "paused"
)

var PoolSCFieldsToRead = []PoolSCField{
	PoolSCFieldC1,
	PoolSCFieldHaircutRate,
	PoolSCFieldPriceOracle,
	PoolsCFieldRetentionRatio,
	PoolSCFieldSlippageParamK,
	PoolSCFieldSlippageParamN,
	PoolSCFieldTokenAddresses,
	PoolSCFieldXThreshold,
	PoolSCFieldPaused,
}

const (
	PoolSCMethodAssetOf           = "assetOf"
	PoolSCMethodGetC1             = "getC1"
	PoolSCMethodGetHaircutRate    = "getHaircutRate"
	PoolSCMethodGetPriceOracle    = "getPriceOracle"
	PoolSCMethodGetRetentionRatio = "getRetentionRatio"
	PoolSCMethodGetSlippageParamK = "getSlippageParamK"
	PoolSCMethodGetSlippageParamN = "getSlippageParamN"
	PoolSCMethodGetTokenAddresses = "getTokenAddresses"
	PoolSCMethodGetXThreshold     = "getXThreshold"
	PoolSCMethodPaused            = "paused"
)

type PoolSCReader struct {
	scanService *service.ScanService
}

func NewPoolSCReader(
	scanService *service.ScanService,
) *PoolSCReader {
	return &PoolSCReader{
		scanService: scanService,
	}
}

func (r *PoolSCReader) Read(
	ctx context.Context,
	address string,
	fields ...PoolSCField,
) (PoolState, error) {
	var calls []*repository.CallParams
	poolState := PoolState{
		Address: address,
	}

	for _, field := range fields {
		switch field {
		case PoolSCFieldC1:
			calls = append(calls, &repository.CallParams{
				ABI:    abis.PlatypusPool,
				Target: address,
				Method: PoolSCMethodGetC1,
				Params: nil,
				Output: &poolState.C1,
			})
		case PoolSCFieldHaircutRate:
			calls = append(calls, &repository.CallParams{
				ABI:    abis.PlatypusPool,
				Target: address,
				Method: PoolSCMethodGetHaircutRate,
				Params: nil,
				Output: &poolState.HaircutRate,
			})
		case PoolSCFieldPriceOracle:
			calls = append(calls, &repository.CallParams{
				ABI:    abis.PlatypusPool,
				Target: address,
				Method: PoolSCMethodGetPriceOracle,
				Params: nil,
				Output: &poolState.PriceOracle,
			})
		case PoolsCFieldRetentionRatio:
			calls = append(calls, &repository.CallParams{
				ABI:    abis.PlatypusPool,
				Target: address,
				Method: PoolSCMethodGetRetentionRatio,
				Params: nil,
				Output: &poolState.RetentionRatio,
			})
		case PoolSCFieldSlippageParamK:
			calls = append(calls, &repository.CallParams{
				ABI:    abis.PlatypusPool,
				Target: address,
				Method: PoolSCMethodGetSlippageParamK,
				Params: nil,
				Output: &poolState.SlippageParamK,
			})
		case PoolSCFieldSlippageParamN:
			calls = append(calls, &repository.CallParams{
				ABI:    abis.PlatypusPool,
				Target: address,
				Method: PoolSCMethodGetSlippageParamN,
				Params: nil,
				Output: &poolState.SlippageParamN,
			})
		case PoolSCFieldTokenAddresses:
			calls = append(calls, &repository.CallParams{
				ABI:    abis.PlatypusPool,
				Target: address,
				Method: PoolSCMethodGetTokenAddresses,
				Params: nil,
				Output: &poolState.TokenAddresses,
			})
		case PoolSCFieldXThreshold:
			calls = append(calls, &repository.CallParams{
				ABI:    abis.PlatypusPool,
				Target: address,
				Method: PoolSCMethodGetXThreshold,
				Params: nil,
				Output: &poolState.XThreshold,
			})
		case PoolSCFieldPaused:
			calls = append(calls, &repository.CallParams{
				ABI:    abis.PlatypusPool,
				Target: address,
				Method: PoolSCMethodPaused,
				Params: nil,
				Output: &poolState.Paused,
			})
		default:
			continue
		}
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
		return poolState, err
	}

	return poolState, nil
}

func (r *PoolSCReader) GetAssetAddresses(
	ctx context.Context,
	address string,
	tokenAddresses []common.Address,
) ([]common.Address, error) {
	assetAddresses := make([]common.Address, len(tokenAddresses))

	calls := make([]*repository.CallParams, 0, len(tokenAddresses))
	for i, tokenAddress := range tokenAddresses {
		calls = append(calls, &repository.CallParams{
			ABI:    abis.PlatypusPool,
			Target: address,
			Method: PoolSCMethodAssetOf,
			Params: []interface{}{tokenAddress},
			Output: &assetAddresses[i],
		})
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
		return nil, err
	}

	return assetAddresses, nil
}

func (r *PoolSCReader) IsPaused(
	ctx context.Context,
	address string,
) (bool, error) {
	state, err := r.Read(ctx, address, PoolSCFieldPaused)
	if err != nil {
		return false, err
	}

	return state.Paused, nil
}
