package gmx

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
)

type USDGReader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewUSDGReader(scanService *service.ScanService) *USDGReader {
	return &USDGReader{
		abi:         abis.ERC20,
		scanService: scanService,
	}
}

func (r *USDGReader) Read(ctx context.Context, address string) (*USDG, error) {
	var totalSupply *big.Int

	err := r.scanService.Call(ctx, &repository.CallParams{
		ABI:    abis.ERC20,
		Target: address,
		Method: USDGMethodTotalSupply,
		Params: nil,
		Output: &totalSupply,
	})
	if err != nil {
		return nil, err
	}

	return &USDG{
		Address:     address,
		TotalSupply: totalSupply,
	}, nil
}
