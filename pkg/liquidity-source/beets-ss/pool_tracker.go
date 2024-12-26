package beets_ss

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
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

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	blocknumber, totalAssets, totalSupply, depositPaused, err := t.getPoolData(ctx, overrides)
	if err != nil {
		return p, err
	}

	return t.updatePool(p, blocknumber, totalAssets, totalSupply, depositPaused)
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

func (d *PoolTracker) getPoolData(ctx context.Context, overrides map[common.Address]gethclient.OverrideAccount) (*big.Int, *big.Int, *big.Int, bool, error) {
	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

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
