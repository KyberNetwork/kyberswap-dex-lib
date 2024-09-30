package unieth

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		ethrpcClient: ethrpcClient,
	}
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
	extra, blockNumber, err := t.getExtra(ctx, overrides)
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

func (t *PoolTracker) getExtra(
	ctx context.Context,
	overrides map[common.Address]gethclient.OverrideAccount,
) (PoolExtra, uint64, error) {
	var (
		paused         bool
		totalSupply    *big.Int
		currentReserve *big.Int
	)

	getPoolStateRequest := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		getPoolStateRequest.SetOverrides(overrides)
	}

	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    StakingABI,
		Target: Staking,
		Method: StakingMethodPaused,
		Params: []interface{}{},
	}, []interface{}{&paused})
	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    StakingABI,
		Target: Staking,
		Method: StakingMethodCurrentReserve,
		Params: []interface{}{},
	}, []interface{}{&currentReserve})
	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    RockXETHABI,
		Target: UNIETH,
		Method: UniETHMethodTotalSupply,
		Params: []interface{}{},
	}, []interface{}{&totalSupply})

	resp, err := getPoolStateRequest.TryAggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}
	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	return PoolExtra{
		Paused:         paused,
		TotalSupply:    totalSupply,
		CurrentReserve: currentReserve,
	}, resp.BlockNumber.Uint64(), nil
}
