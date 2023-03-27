package makerpsm

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
)

type VatReader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewVatReader(scanService *service.ScanService) *VatReader {
	return &VatReader{
		abi:         abis.MakerPSMVat,
		scanService: scanService,
	}
}

func (r *VatReader) Read(ctx context.Context, address string, ilk [32]byte) (*Vat, error) {
	vat := Vat{
		ILK: ILK{},
	}

	calls := []*repository.CallParams{
		{
			ABI:    r.abi,
			Target: address,
			Method: VatMethodDebt,
			Params: nil,
			Output: &vat.Debt,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VatMethodLine,
			Params: nil,
			Output: &vat.Line,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VatMethodIlks,
			Params: []interface{}{ilk},
			Output: &vat.ILK,
		},
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
		return nil, err
	}

	return &vat, nil
}
