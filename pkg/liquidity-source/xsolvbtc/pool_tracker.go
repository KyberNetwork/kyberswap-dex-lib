package xsolvbtc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	extra := &PoolExtra{}
	if err := json.Unmarshal([]byte(p.Extra), extra); err != nil {
		return p, err
	}

	blockNumber, err := updateExtra(ctx, extra, t.cfg, t.ethrpcClient)
	if err != nil {
		return p, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()

	return p, nil
}
