package madmex

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
)

type PancakePairReader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewPancakePairReader(scanService *service.ScanService) *PancakePairReader {
	return &PancakePairReader{
		abi:         abis.GMXPancakePair,
		scanService: scanService,
	}
}

func (r *PancakePairReader) Read(ctx context.Context, address string) (*PancakePair, error) {
	var reserves struct {
		Reserve0           *big.Int
		Reserve1           *big.Int
		BlockTimestampLast uint32
	}

	err := r.scanService.Call(ctx, &repository.CallParams{
		ABI:    r.abi,
		Target: address,
		Method: PancakePairMethodGetReserves,
		Output: &reserves,
	})
	if err != nil {
		return nil, err
	}

	return &PancakePair{
		Reserves: []*big.Int{
			reserves.Reserve0,
			reserves.Reserve1,
		},
		TimestampLast: reserves.BlockTimestampLast,
	}, nil
}
