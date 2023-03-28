package metavault

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
)

type USDMReader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewUSDMReader(scanService *service.ScanService) *USDMReader {
	return &USDMReader{
		abi:         abis.ERC20,
		scanService: scanService,
	}
}

func (r *USDMReader) Read(ctx context.Context, address string) (*USDM, error) {
	var totalSupply *big.Int

	err := r.scanService.Call(ctx, &repository.CallParams{
		ABI:    r.abi,
		Target: address,
		Method: USDMMethodTotalSupply,
		Params: nil,
		Output: &totalSupply,
	})
	if err != nil {
		return nil, err
	}

	return &USDM{
		Address:     address,
		TotalSupply: totalSupply,
	}, nil
}
