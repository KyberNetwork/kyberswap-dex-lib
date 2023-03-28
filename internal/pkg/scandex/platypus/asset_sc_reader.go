package platypus

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
)

type AssetSCField string

const (
	AssetSCFieldCash             = "cash"
	AssetSCFieldDecimals         = "decimals"
	AssetSCFieldLiability        = "liability"
	AssetSCFieldUnderlyingToken  = "underlyingToken"
	AssetSCFieldAggregateAccount = "aggregateAccount"
)

var AssetSCFieldsToRead = []AssetSCField{
	AssetSCFieldCash,
	AssetSCFieldDecimals,
	AssetSCFieldLiability,
	AssetSCFieldUnderlyingToken,
	AssetSCFieldAggregateAccount,
}

const (
	AssetSCMethodCash             = "cash"
	AssetSCMethodDecimals         = "decimals"
	AssetSCMethodLiability        = "liability"
	AssetSCMethodUnderlyingToken  = "underlyingToken"
	AssetSCMethodAggregateAccount = "aggregateAccount"
)

type AssetSCReader struct {
	scanService *service.ScanService
}

func NewAssetSCReader(
	scanService *service.ScanService,
) *AssetSCReader {
	return &AssetSCReader{
		scanService: scanService,
	}
}

func (r *AssetSCReader) BulkRead(
	ctx context.Context,
	addresses []string,
	fields ...AssetSCField,
) ([]AssetState, error) {
	assetStates := make([]AssetState, len(addresses))
	var calls []*repository.CallParams

	for i, address := range addresses {
		assetStates[i] = AssetState{
			Address: address,
		}

		for _, field := range fields {
			switch field {
			case AssetSCFieldCash:
				calls = append(calls, &repository.CallParams{
					ABI:    abis.PlatypusAsset,
					Target: address,
					Method: AssetSCMethodCash,
					Params: nil,
					Output: &assetStates[i].Cash,
				})
			case AssetSCFieldDecimals:
				calls = append(calls, &repository.CallParams{
					ABI:    abis.PlatypusAsset,
					Target: address,
					Method: AssetSCMethodDecimals,
					Params: nil,
					Output: &assetStates[i].Decimals,
				})
			case AssetSCFieldLiability:
				calls = append(calls, &repository.CallParams{
					ABI:    abis.PlatypusAsset,
					Target: address,
					Method: AssetSCMethodLiability,
					Params: nil,
					Output: &assetStates[i].Liability,
				})
			case AssetSCFieldUnderlyingToken:
				calls = append(calls, &repository.CallParams{
					ABI:    abis.PlatypusAsset,
					Target: address,
					Method: AssetSCMethodUnderlyingToken,
					Params: nil,
					Output: &assetStates[i].UnderlyingToken,
				})
			case AssetSCFieldAggregateAccount:
				calls = append(calls, &repository.CallParams{
					ABI:    abis.PlatypusAsset,
					Target: address,
					Method: AssetSCMethodAggregateAccount,
					Params: nil,
					Output: &assetStates[i].AggregateAccount,
				})
			default:
				continue
			}
		}
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
		return nil, err
	}

	return assetStates, nil
}
