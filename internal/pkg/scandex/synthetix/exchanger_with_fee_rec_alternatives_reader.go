package synthetix

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
)

// ExchangerWithFeeRecAlternatives methods
const (
	ExchangerWithFeeRecAlternativesMethodAtomicMaxVolumePerBlock = "atomicMaxVolumePerBlock"
	ExchangerWithFeeRecAlternativesMethodLastAtomicVolume        = "lastAtomicVolume"
)

type ExchangerWithFeeRecAlternativesReader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewExchangerWithFeeRecAlternativesReader(scanService *service.ScanService) *ExchangerWithFeeRecAlternativesReader {
	return &ExchangerWithFeeRecAlternativesReader{
		abi:         abis.SynthetixExchangerWithFeeRecAlternatives,
		scanService: scanService,
	}
}

func (r *ExchangerWithFeeRecAlternativesReader) Read(
	ctx context.Context,
	poolState *PoolState,
) (*PoolState, error) {
	if err := r.readData(ctx, poolState); err != nil {
		return nil, err
	}

	return poolState, nil
}

// readData reads data which required no parameters, included:
// - AtomicMaxVolumePerBlock
// - LastAtomicVolume
func (r *ExchangerWithFeeRecAlternativesReader) readData(
	ctx context.Context,
	poolState *PoolState,
) error {
	address := poolState.Addresses.Exchanger

	var lastAtomicVolume ExchangeVolumeAtPeriod

	calls := []*repository.CallParams{
		{
			ABI:    r.abi,
			Target: address,
			Method: ExchangerWithFeeRecAlternativesMethodAtomicMaxVolumePerBlock,
			Params: nil,
			Output: &poolState.AtomicMaxVolumePerBlock,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: ExchangerWithFeeRecAlternativesMethodLastAtomicVolume,
			Params: nil,
			Output: &lastAtomicVolume,
		},
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
		return err
	}

	poolState.LastAtomicVolume = &lastAtomicVolume

	return nil
}
