package maverick

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/logger"
	"math/big"
	"strconv"
	"time"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)

	var (
		fee, binBalanceA, binBalanceB, tokenAScale, tokenBScale *big.Int
		getStateResult                                          GetStateResult
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodFee,
		Params: nil,
	}, []interface{}{&fee})

	calls.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodGetState,
		Params: nil,
	}, []interface{}{&getStateResult})

	calls.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodBinBalanceA,
		Params: nil,
	}, []interface{}{&binBalanceA})

	calls.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodBinBalanceB,
		Params: nil,
	}, []interface{}{&binBalanceB})

	calls.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodTokenAScale,
		Params: nil,
	}, []interface{}{&tokenAScale})

	calls.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodTokenBScale,
		Params: nil,
	}, []interface{}{&tokenBScale})

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to aggregate to get pool data")

		return entity.Pool{}, err
	}

	if getStateResult.BinCounter == nil || getStateResult.ActiveTick == nil {
		err := fmt.Errorf("binCounter or activeTick must not nil")
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"err":         err,
		}).Errorf("failed to aggregate to get pool data")

		return entity.Pool{}, err
	}
	binCounter := int(getStateResult.BinCounter.Int64())
	activeTick := getStateResult.ActiveTick

	var bins = make(map[string]Bin, binCounter)
	for i := 0; i < binCounter; i++ {
		calls.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: poolMethodGetBin,
			Params: nil,
		}, []interface{}{bins[strconv.Itoa(i)]})
	}
	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to aggregate to get pool data")

		return entity.Pool{}, err
	}

	var binPositions = make(map[string]map[string]*big.Int)
	var binMap = make(map[string]*big.Int)
	var binMapIndex = d.getBinMapIndex(activeTick)
	for _, bin := range bins {
		calls.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: poolMethodBinPositions,
			Params: []interface{}{bin.LowerTick.String(), bin.Kind.String()},
		}, []interface{}{binPositions[bin.LowerTick.String()][bin.Kind.String()]})
	}
	calls.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodBinMap,
		Params: []interface{}{binMapIndex},
	}, []interface{}{binMap[binMapIndex.String()]})

	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to try aggregate to get pool data")

		return entity.Pool{}, err
	}

	var extra = Extra{
		Fee:              fee,
		ProtocolFeeRatio: getStateResult.ProtocolFeeRatio,
		ActiveTick:       getStateResult.ActiveTick,
		BinCounter:       getStateResult.BinCounter,
		Bins:             bins,
		BinPositions:     binPositions,
		BinMap:           binMap,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to marshal extra")
		return entity.Pool{}, err
	}

	p.Reserves = entity.PoolReserves{binBalanceA.String(), binBalanceB.String()}
	p.Timestamp = time.Now().Unix()
	p.Extra = string(extraBytes)

	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}

func (d *PoolTracker) getBinMapIndex(activeTick *big.Int) *big.Int {
	mapIndex := new(big.Int).Rsh(activeTick, 8)

	return mapIndex
}
