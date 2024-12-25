package beets_ss

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	blocknumber, totalAssets, totalSupply, depositPaused, err := d.getPoolData(ctx)
	if err != nil {
		return p, err
	}

	return d.updatePool(p, blocknumber, totalAssets, totalSupply, depositPaused)
}

func (d *PoolTracker) updatePool(pool entity.Pool, blocknumber, totalAssets, totalSupply *big.Int, depositPaused bool) (entity.Pool, error) {
	extra := Extra{
		TotalAssets:   totalAssets,
		TotalSupply:   totalSupply,
		DepositPaused: depositPaused,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return pool, err
	}

	pool.Extra = string(extraBytes)
	pool.BlockNumber = blocknumber.Uint64()
	pool.Timestamp = time.Now().Unix()

	return pool, nil
}

func (d *PoolTracker) getPoolData(ctx context.Context) (*big.Int, *big.Int, *big.Int, bool, error) {
	req := d.ethrpcClient.NewRequest().SetContext(ctx)

	var (
		totalAssets   = ZERO
		totalSupply   = ZERO
		depositPaused bool
	)

	req.AddCall(&ethrpc.Call{
		ABI:    sonicStakingABI,
		Target: Beets_Staked_Sonic_Address,
		Method: methodTotalAssets,
		Params: nil,
	}, []interface{}{&totalAssets})
	req.AddCall(&ethrpc.Call{
		ABI:    sonicStakingABI,
		Target: Beets_Staked_Sonic_Address,
		Method: methodTotalSupply,
		Params: nil,
	}, []interface{}{&totalSupply})
	req.AddCall(&ethrpc.Call{
		ABI:    sonicStakingABI,
		Target: Beets_Staked_Sonic_Address,
		Method: methodDepositPaused,
		Params: nil,
	}, []interface{}{&depositPaused})

	resp, err := req.Aggregate()
	if err != nil {
		return nil, nil, nil, false, err
	}

	return resp.BlockNumber, totalAssets, totalSupply, depositPaused, nil
}
