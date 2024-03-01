package savingsdai

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{ethrpcClient: ethrpcClient}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"dexType":     DexType,
		"poolAddress": p.Address,
	}).Info("Start updating state ...")

	defer func() {
		logger.WithFields(logger.Fields{
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Info("Finish updating state.")
	}()

	var (
		dsr, rho, chi            *big.Int
		totalAssets, totalSupply *big.Int
	)

	req := t.ethrpcClient.R().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    potABI,
		Target: pot,
		Method: potMethodDSR,
	}, []interface{}{&dsr})

	req.AddCall(&ethrpc.Call{
		ABI:    potABI,
		Target: pot,
		Method: potMethodRHO,
	}, []interface{}{&rho})

	req.AddCall(&ethrpc.Call{
		ABI:    potABI,
		Target: pot,
		Method: potMethodCHI,
	}, []interface{}{&chi})

	req.AddCall(&ethrpc.Call{
		ABI:    savingsdaiABI,
		Target: Savingsdai,
		Method: savingsdaiMethodTotalAssets,
	}, []interface{}{&totalAssets})

	req.AddCall(&ethrpc.Call{
		ABI:    savingsdaiABI,
		Target: Savingsdai,
		Method: savingsdaiMethodTotalSupply,
	}, []interface{}{&totalSupply})

	result, err := req.Aggregate()
	if err != nil {
		return p, err
	}

	req = t.ethrpcClient.R().SetContext(ctx)
	blockTimestamp, err := req.GetCurrentBlockTimestamp()
	if err != nil {
		return p, err
	}

	dsr_, _ := uint256.FromBig(dsr)
	rho_, _ := uint256.FromBig(rho)
	chi_, _ := uint256.FromBig(chi)

	extraBytes, err := json.Marshal(Extra{
		BlockTimestamp: uint256.NewInt(blockTimestamp + blocktime),
		DSR:            dsr_,
		RHO:            rho_,
		CHI:            chi_,
	})
	if err != nil {
		return p, err
	}

	p.Reserves = entity.PoolReserves{totalAssets.String(), totalSupply.String()}
	p.Timestamp = time.Now().Unix()
	p.Extra = string(extraBytes)
	p.BlockNumber = result.BlockNumber.Uint64()

	return p, nil
}
