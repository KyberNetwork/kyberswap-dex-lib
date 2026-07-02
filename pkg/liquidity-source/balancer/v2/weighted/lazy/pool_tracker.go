package lazy

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v2/shared"
	weighted "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v2/weighted"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	poolTypeVer1 = 1

	poolMethodGetSwapFeePercentage = "getSwapFeePercentage"
	poolMethodGetPausedState       = "getPausedState"
	poolMethodTotalSupply          = "totalSupply"
	poolMethodGetInvariant         = "getInvariant"
	poolMethodGetLastInvariant     = "getLastInvariant"

	protocolMethodGetSwapFeePercentage = "getSwapFeePercentage"
)

type PoolTracker struct {
	config       *shared.Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(weighted.DexType, NewPoolTracker)

func NewPoolTracker(
	config *shared.Config,
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
	_ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"dexId":       t.config.DexID,
		"dexType":     weighted.DexType,
		"poolAddress": p.Address,
	}).Info("Start updating state ...")

	defer func() {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     weighted.DexType,
			"poolAddress": p.Address,
		}).Info("Finish updating state.")
	}()

	var staticExtra weighted.StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     weighted.DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())

		return p, err
	}

	d := newTrackerData()
	req := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides)
	addRPCCalls(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) },
		p.Address, staticExtra.PoolTypeVer, staticExtra.PoolID, staticExtra.Vault,
		t.config.ProtocolFeesCollector, d)

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     weighted.DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())

		return p, err
	}

	return buildPoolState(t, p, d, res.BlockNumber)
}

type trackerData struct {
	poolTokens                weighted.PoolTokens
	swapFeePercentage         *big.Int
	protocolSwapFeePercentage *big.Int
	pausedState               weighted.PausedState
	lastInvariant             *big.Int
	totalSupply               *big.Int
}

func newTrackerData() *trackerData {
	return &trackerData{
		protocolSwapFeePercentage: bignumber.ZeroBI,
		lastInvariant:             bignumber.ZeroBI,
	}
}

func addRPCCalls(addFn func(*ethrpc.Call, []any), poolAddress string, poolTypeVer int,
	poolID, vault, protocolFeesCollector string, d *trackerData) {
	addFn(&ethrpc.Call{
		ABI:    shared.VaultABI,
		Target: vault,
		Method: shared.VaultMethodGetPoolTokens,
		Params: []any{common.HexToHash(poolID)},
	}, []any{&d.poolTokens})
	addFn(&ethrpc.Call{
		ABI:    weighted.PoolABI,
		Target: poolAddress,
		Method: poolMethodGetSwapFeePercentage,
	}, []any{&d.swapFeePercentage})
	addFn(&ethrpc.Call{
		ABI:    weighted.PoolABI,
		Target: poolAddress,
		Method: poolMethodGetPausedState,
	}, []any{&d.pausedState})
	if poolTypeVer == poolTypeVer1 {
		addFn(&ethrpc.Call{
			ABI:    weighted.PoolABI,
			Target: poolAddress,
			Method: poolMethodGetLastInvariant,
		}, []any{&d.lastInvariant})
	} else {
		addFn(&ethrpc.Call{
			ABI:    weighted.PoolABI,
			Target: poolAddress,
			Method: poolMethodGetInvariant,
		}, []any{&d.lastInvariant})
	}
	addFn(&ethrpc.Call{
		ABI:    weighted.PoolABI,
		Target: poolAddress,
		Method: poolMethodTotalSupply,
	}, []any{&d.totalSupply})
	if protocolFeesCollector != "" {
		addFn(&ethrpc.Call{
			ABI:    shared.ProtocolFeesCollectorABI,
			Target: protocolFeesCollector,
			Method: protocolMethodGetSwapFeePercentage,
		}, []any{&d.protocolSwapFeePercentage})
	}
}

func buildPoolState(t *PoolTracker, p entity.Pool, d *trackerData, blockNumber *big.Int) (entity.Pool, error) {
	extra := weighted.Extra{
		SwapFeePercentage:         uint256.MustFromBig(d.swapFeePercentage),
		ProtocolSwapFeePercentage: uint256.MustFromBig(d.protocolSwapFeePercentage),
		LastInvariant:             uint256.MustFromBig(d.lastInvariant),
		TotalSupply:               uint256.MustFromBig(d.totalSupply),
		Paused:                    !isNotPaused(d.pausedState),
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     weighted.DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())
		return p, err
	}

	reserves, err := initReserves(t, p, d.poolTokens)
	if err != nil {
		return p, err
	}

	if blockNumber != nil {
		p.BlockNumber = blockNumber.Uint64()
	}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves

	return p, nil
}

func initReserves(t *PoolTracker, p entity.Pool, poolTokens weighted.PoolTokens) ([]string, error) {
	reserveByToken := make(map[string]*big.Int)
	for idx, token := range poolTokens.Tokens {
		addr := hexutil.Encode(token[:])
		reserveByToken[addr] = poolTokens.Balances[idx]
	}

	reserves := make([]string, len(p.Tokens))
	for idx, token := range p.Tokens {
		r, ok := reserveByToken[token.Address]
		if !ok {
			logger.WithFields(logger.Fields{
				"dexId":       t.config.DexID,
				"dexType":     weighted.DexType,
				"poolAddress": p.Address,
			}).Error("can not get reserve")
			return nil, weighted.ErrReserveNotFound
		}
		reserves[idx] = r.String()
	}

	return reserves, nil
}

func isNotPaused(pausedState weighted.PausedState) bool {
	return time.Now().Unix() > pausedState.BufferPeriodEndTime.Int64() || !pausedState.Paused
}
