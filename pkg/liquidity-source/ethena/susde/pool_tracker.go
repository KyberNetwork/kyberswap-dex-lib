package susde

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

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
		totalAssets *big.Int
		totalSupply *big.Int
	)

	req := t.ethrpcClient.R().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    stakedUSDeV2ABI,
		Target: StakedUSDeV2,
		Method: stakedUSDeV2MethodTotalSupply,
	}, []interface{}{&totalSupply})

	req.AddCall(&ethrpc.Call{
		ABI:    stakedUSDeV2ABI,
		Target: StakedUSDeV2,
		Method: stakedUSDeV2MethodTotalAssets,
	}, []interface{}{&totalAssets})

	result, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())
		return p, err
	}

	p.Reserves = entity.PoolReserves{
		totalAssets.String(),
		totalSupply.String(),
	}
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = result.BlockNumber.Uint64()

	return p, nil
}
