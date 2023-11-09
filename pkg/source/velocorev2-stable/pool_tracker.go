package velocorev2stable

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"dexID":   d.cfg.DexID,
		"dexType": DexTypeVelocoreV2Stable,
		"address": p.Address,
	}).Infof("Start getting new state of pool")

	var poolDataResp poolDataResp

	req := d.ethrpcClient.R()

	req.AddCall(&ethrpc.Call{
		ABI:    lensABI,
		Target: d.cfg.LensAddress,
		Method: lensMethodQueryPool,
		Params: []interface{}{common.HexToAddress(p.Address)},
	}, []interface{}{&poolDataResp})

	poolDat := newPoolData(poolDataResp)

}
