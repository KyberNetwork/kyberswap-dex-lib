package dodo

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	cmap "github.com/orcaman/concurrent-map"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	blackList    cmap.ConcurrentMap
}

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	blackList, err := initBlackList(cfg.BlacklistFilePath)
	if err != nil {
		return nil, err
	}

	return &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
		blackList:    blackList,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	var staticExtraData = struct {
		Type string `json:"type"`
	}{}
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtraData); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to unmarshal extra data")
		return entity.Pool{}, err
	}
	if staticExtraData.Type == subgraphPoolTypeDodoClassical {
		return d.getNewPoolStateDodoV1(ctx, p)
	}

	return d.getNewPoolStateDodoV2(ctx, p)
}

func (d *PoolTracker) getNewPoolStateDodoV1(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.Infof("[Dodo] Start getting new state of dodoV1 pool: %v", p.Address)

	var (
		targetReserve                                         TargetReserve
		i, k, lpFeeRate, mtFeeRate, baseReserve, quoteReserve *big.Int
		rStatus                                               uint8
		tradeAllow                                            bool
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    v1PoolABI,
		Target: p.Address,
		Method: poolMethodGetExpectedTarget,
		Params: nil,
	}, []interface{}{&targetReserve})

	calls.AddCall(&ethrpc.Call{
		ABI:    v1PoolABI,
		Target: p.Address,
		Method: poolMethodK,
		Params: nil,
	}, []interface{}{&k})

	calls.AddCall(&ethrpc.Call{
		ABI:    v1PoolABI,
		Target: p.Address,
		Method: poolMethodRStatus,
		Params: nil,
	}, []interface{}{&rStatus})

	calls.AddCall(&ethrpc.Call{
		ABI:    v1PoolABI,
		Target: p.Address,
		Method: poolMethodGetOraclePrice,
		Params: nil,
	}, []interface{}{&i})

	calls.AddCall(&ethrpc.Call{
		ABI:    v1PoolABI,
		Target: p.Address,
		Method: poolMethodLpFeeRate,
		Params: nil,
	}, []interface{}{&lpFeeRate})

	calls.AddCall(&ethrpc.Call{
		ABI:    v1PoolABI,
		Target: p.Address,
		Method: poolMethodMtFeeRate,
		Params: nil,
	}, []interface{}{&mtFeeRate})

	calls.AddCall(&ethrpc.Call{
		ABI:    v1PoolABI,
		Target: p.Address,
		Method: poolMethodBaseBalance,
		Params: nil,
	}, []interface{}{&baseReserve})

	calls.AddCall(&ethrpc.Call{
		ABI:    v1PoolABI,
		Target: p.Address,
		Method: poolMethodQuoteBalance,
		Params: nil,
	}, []interface{}{&quoteReserve})

	calls.AddCall(&ethrpc.Call{
		ABI:    v1PoolABI,
		Target: p.Address,
		Method: poolMethodTradeAllowed,
		Params: nil,
	}, []interface{}{&tradeAllow})

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("[DodoV1] failed to aggregate for pool data")

		return entity.Pool{}, err
	}

	extra := Extra{
		I:              i,
		K:              k,
		RStatus:        int(rStatus),
		MtFeeRate:      new(big.Float).Quo(new(big.Float).SetInt64(mtFeeRate.Int64()), oneBF),
		LpFeeRate:      new(big.Float).Quo(new(big.Float).SetInt64(lpFeeRate.Int64()), oneBF),
		Swappable:      true,
		Reserves:       []*big.Int{baseReserve, quoteReserve},
		TargetReserves: []*big.Int{targetReserve.BaseTarget, targetReserve.QuoteTarget},
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to marshaling the extra bytes data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.SwapFee = bigToFloat64(extra.MtFeeRate) + bigToFloat64(extra.LpFeeRate)
	p.Reserves = entity.PoolReserves{baseReserve.String(), quoteReserve.String()}

	logger.Infof("[Dodo] Finish getting new state of dodoV1 pool: %v", p.Address)

	return p, nil
}

func (d *PoolTracker) getNewPoolStateDodoV2(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.Infof("[Dodo] Start getting new state of dodoV2 pool: %v", p.Address)

	_, ok := d.blackList.Get(p.Address)
	if ok {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
		}).Error(ErrPoolAddressBanned.Error())

		return entity.Pool{}, ErrPoolAddressBanned
	}

	var (
		state     PoolState
		feeRate   FeeRate
		lpFeeRate *big.Int
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    v2PoolABI,
		Target: p.Address,
		Method: poolMethodGetPMMStateForCall,
		Params: nil,
	}, []interface{}{&state})

	calls.AddCall(&ethrpc.Call{
		ABI:    v2PoolABI,
		Target: p.Address,
		Method: poolMethodLpFeeRate,
		Params: nil,
	}, []interface{}{&lpFeeRate})

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("[DodoV2] failed to aggregate for pool data")
		return entity.Pool{}, err
	}

	// Some DPP pools have an issue with `getUserFeeRate` function, so we need to separately call
	calls = d.ethrpcClient.NewRequest()
	calls.AddCall(&ethrpc.Call{
		ABI:    v2PoolABI,
		Target: p.Address,
		Method: poolMethodGetUserFeeRate,
		Params: []interface{}{common.HexToAddress(p.Address)},
	}, []interface{}{&feeRate})
	if _, err := calls.Call(); err != nil {
		// retry 1 time before adding to blacklist
		if _, errRetry := calls.Call(); errRetry != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
				"error":       errRetry,
			}).Errorf("[DodoV2] failed to call getUserFeeRate, add pool address to blacklist")
			d.blackList.Set(p.Address, true)

			return entity.Pool{}, err
		}
	}

	if state.B == nil && state.Q == nil &&
		state.B0 == nil && state.Q0 == nil &&
		state.I == nil && state.K == nil && state.R == nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
		}).Errorf("get pool state failed")

		return entity.Pool{}, fmt.Errorf("get pool state failed")
	}

	if feeRate.MtFeeRate == nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
		}).Errorf("get pool feeRate failed")

		return entity.Pool{}, fmt.Errorf("get pool feeRate failed")
	}

	if lpFeeRate == nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
		}).Errorf("get pool lpFeeRate failed")

		return entity.Pool{}, fmt.Errorf("get pool lpFeeRate failed")
	}

	extra := Extra{
		I:              state.I,
		K:              state.K,
		RStatus:        int(state.R.Int64()),
		MtFeeRate:      new(big.Float).Quo(new(big.Float).SetInt64(feeRate.MtFeeRate.Int64()), oneBF),
		LpFeeRate:      new(big.Float).Quo(new(big.Float).SetInt64(lpFeeRate.Int64()), oneBF),
		Swappable:      true,
		Reserves:       []*big.Int{state.B, state.Q},
		TargetReserves: []*big.Int{state.B0, state.Q0},
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to marshaling the extra bytes data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.SwapFee = bigToFloat64(extra.LpFeeRate) + bigToFloat64(extra.MtFeeRate)
	p.Reserves = entity.PoolReserves{state.B.String(), state.Q.String()}
	p.Timestamp = time.Now().Unix()

	logger.Infof("[Dodo] Finish updating state of dodoV2 pool: %v", p.Address)

	return p, nil
}

func initBlackList(blackListPath string) (cmap.ConcurrentMap, error) {
	blackListMap := cmap.New()

	if blackListPath == "" {
		return blackListMap, nil
	}

	byteData, ok := bytesByPath[blackListPath]
	if !ok {
		logger.WithFields(logger.Fields{
			"blacklistFilePath": blackListPath,
		}).Error(ErrInitializeBlacklistFailed.Error())

		return blackListMap, ErrInitializeBlacklistFailed
	}

	file := bytes.NewReader(byteData)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		poolAddress := scanner.Text()
		if poolAddress != "" {
			blackListMap.Set(poolAddress, true)
		}
	}

	return blackListMap, nil
}

func bigToFloat64(b *big.Float) float64 {
	f, _ := b.Float64()

	return f
}
