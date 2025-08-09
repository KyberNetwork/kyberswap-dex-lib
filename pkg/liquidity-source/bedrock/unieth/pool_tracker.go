package unieth

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryE0(DexType, NewPoolTracker)

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
	var extra PoolExtra

	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).AddCall(&ethrpc.Call{
		ABI:    StakingABI,
		Target: Staking,
		Method: StakingMethodPaused,
	}, []any{&extra.Paused}).AddCall(&ethrpc.Call{
		ABI:    StakingABI,
		Target: Staking,
		Method: StakingMethodCurrentReserve,
	}, []any{&extra.CurrentReserve}).AddCall(&ethrpc.Call{
		ABI:    RockXETHABI,
		Target: UNIETH,
		Method: UniETHMethodTotalSupply,
	}, []any{&extra.TotalSupply}).TryBlockAndAggregate()
	if err != nil {
		return extra, 0, err
	} else if resp.BlockNumber == nil {
		resp.BlockNumber = bignumber.ZeroBI
	}

	return extra, resp.BlockNumber.Uint64(), nil
}
