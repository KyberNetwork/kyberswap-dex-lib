package makerpsm

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
)

type PSMReader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewPSMReader(scanService *service.ScanService) *PSMReader {
	return &PSMReader{
		abi:         abis.MakerPSMPSM,
		scanService: scanService,
	}
}

func (r *PSMReader) Read(ctx context.Context, address string) (*PSM, error) {
	psm := PSM{}

	calls := []*repository.CallParams{
		{
			ABI:    r.abi,
			Target: address,
			Method: PSMMethodTIn,
			Params: nil,
			Output: &psm.TIn,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: PSMMethodTOut,
			Params: nil,
			Output: &psm.TOut,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: PSMMethodVat,
			Params: nil,
			Output: &psm.VatAddress,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: PSMMethodIlk,
			Params: nil,
			Output: &psm.ILK,
		},
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
		return nil, err
	}

	return &psm, nil
}
