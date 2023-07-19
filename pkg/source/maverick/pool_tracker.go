package maverick

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/logger"
	"math"
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

	binCounter := getStateResult.State.BinCounter
	activeTick := big.NewInt(int64(getStateResult.State.ActiveTick))
	protocolFeeRatio := big.NewInt(int64(getStateResult.State.ProtocolFeeRatio))

	binLength := int(binCounter.Int64())
	binList := make([]GetBinResult, binLength)

	binCalls := d.ethrpcClient.NewRequest().SetContext(ctx)
	for i := 0; i < binLength; i++ {
		binCalls.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: poolMethodGetBin,
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&binList[i]})
	}
	if _, err := binCalls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to aggregate to get pool data")

		return entity.Pool{}, err
	}

	bins := d.binListToMap(binList)

	minTick := math.MaxInt
	maxTick := -math.MaxInt
	for _, bin := range bins {
		binTick := int(bin.LowerTick.Int64())
		if minTick > binTick {
			minTick = binTick
		}
		if maxTick < binTick {
			maxTick = binTick
		}
	}
	if minTick < 0 {
		minTick *= -1
	}

	binMapPositive := make([]*big.Int, maxTick+1)
	binMapNegative := make([]*big.Int, minTick+1)

	binMapCalls := d.ethrpcClient.NewRequest().SetContext(ctx)
	for _, bin := range bins {
		binMapIndex := d.getBinMapIndex(bin.LowerTick)
		fmt.Println("---binMapIndex", binMapIndex)
		if binMapIndex.Sign() >= 0 {
			binMapCalls.AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: p.Address,
				Method: poolMethodBinMap,
				Params: []interface{}{int32(binMapIndex.Int64())},
			}, []interface{}{&binMapPositive[int(binMapIndex.Int64())]})
		} else {
			binMapCalls.AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: p.Address,
				Method: poolMethodBinMap,
				Params: []interface{}{int32(binMapIndex.Int64())},
			}, []interface{}{&binMapNegative[-int(binMapIndex.Int64())]})
		}
	}
	if _, err := binMapCalls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to try aggregate to get pool data")

		return entity.Pool{}, err
	}

	binMap := make(map[string]*big.Int, 0)
	for i, b := range binMapPositive {
		if b != nil {
			binMap[strconv.Itoa(i)] = new(big.Int).Set(b)
		}
	}
	for i, b := range binMapNegative {
		if b != nil {
			binMap[strconv.Itoa(-i)] = new(big.Int).Set(b)
		}
	}

	var extra = Extra{
		Fee:              fee,
		ProtocolFeeRatio: protocolFeeRatio,
		ActiveTick:       activeTick,
		BinCounter:       binCounter,
		Bins:             bins,
		BinPositions:     d.generateBinPosition(bins),
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

func (d *PoolTracker) binListToMap(binList []GetBinResult) map[string]Bin {
	bins := make(map[string]Bin, len(binList))
	for i, bin := range binList {
		strI := strconv.Itoa(i)
		bins[strI] = Bin{
			ReserveA:  bin.BinState.ReserveA,
			ReserveB:  bin.BinState.ReserveB,
			LowerTick: big.NewInt(int64(bin.BinState.LowerTick)),
			Kind:      big.NewInt(int64(bin.BinState.Kind)),
		}
	}

	return bins
}

func (d *PoolTracker) generateBinPosition(bins map[string]Bin) map[string]map[string]*big.Int {
	binPositions := make(map[string]map[string]*big.Int, 0)
	for i, bin := range bins {
		if binPositions[bin.LowerTick.String()] == nil {
			binPositions[bin.LowerTick.String()] = make(map[string]*big.Int)
		}
		binPositions[bin.LowerTick.String()][bin.Kind.String()] = bignumber.NewBig10(i)
	}

	return binPositions
}
