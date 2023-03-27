package platypus

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
)

const StakedAvaxSCMethodGetPooledAvaxByShares = "getPooledAvaxByShares"

type StakedAvaxSCReader struct {
	scanService *service.ScanService
}

func NewStakedAvaxSCReader(
	scanService *service.ScanService,
) *StakedAvaxSCReader {
	return &StakedAvaxSCReader{
		scanService: scanService,
	}
}

func (r *StakedAvaxSCReader) GetSAvaxRate(ctx context.Context, address string) (*big.Int, error) {
	var sAvaxRate *big.Int

	err := r.scanService.Call(ctx, &repository.CallParams{
		ABI:    abis.StakedAvax,
		Target: address,
		Method: StakedAvaxSCMethodGetPooledAvaxByShares,
		Params: []interface{}{constant.BONE},
		Output: &sAvaxRate,
	})
	if err != nil {
		return nil, err
	}

	return sAvaxRate, nil
}
