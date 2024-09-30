package maverickv1

import (
	"context"
	"encoding/json"
	"math/big"
	"strconv"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/sourcegraph/conc/pool"
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

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, params, nil)
}

func (d *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params sourcePool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, sourcePool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (d *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ sourcePool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)

	var (
		fee, binBalanceA, binBalanceB, tokenAScale, tokenBScale *big.Int
		getStateResult                                          GetStateResult
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}

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
	binRaws := make([]GetBinResult, binLength+1)

	// NOTE:
	// binLength of pool 0xd0b2f5018b5d22759724af6d4281ac0b13266360 can reach 2751, cause entity too large error when using multicall
	// split bins into chunk to get concurrency
	chunk := d.config.GetBinChunk
	if chunk == 0 {
		chunk = defaultChunk
	}
	g := pool.New().WithContext(ctx)
	for i := 0; i <= binLength; i += chunk {
		startBin := i
		endBin := startBin + chunk - 1
		if endBin > binLength {
			endBin = binLength
		}
		g.Go(func(context.Context) error {
			return func(startBin, endBin int) error {
				binCalls := d.ethrpcClient.NewRequest().SetContext(ctx)
				if overrides != nil {
					binCalls.SetOverrides(overrides)
				}
				for j := startBin; j <= endBin; j++ {
					binCalls.AddCall(&ethrpc.Call{
						ABI:    poolABI,
						Target: p.Address,
						Method: poolMethodGetBin,
						Params: []interface{}{big.NewInt(int64(j))},
					}, []interface{}{&binRaws[j]})
				}
				if _, err := binCalls.Aggregate(); err != nil {
					logger.WithFields(logger.Fields{
						"poolAddress": p.Address,
						"error":       err,
						"startBin":    startBin,
						"endBin":      endBin,
					}).Errorf("failed to aggregate to get bins data")

					return err
				}
				return nil
			}(startBin, endBin)
		})
	}
	if err := g.Wait(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to aggregate to get pool data")

		return entity.Pool{}, err
	}

	// Generate bins, binPosition, binMap from binRaws
	bins := make(map[string]Bin)
	binPositions := make(map[string]map[string]*big.Int)
	binMap := make(map[string]*big.Int)
	binMapHex := make(map[string]*big.Int)
	var minBinMapIndex, maxBinMapIndex *big.Int
	for i, binRaw := range binRaws {
		if binRaw.BinState.MergeID.Cmp(zeroBI) != 0 ||
			(binRaw.BinState.ReserveA.Cmp(zeroBI) == 0 && binRaw.BinState.ReserveB.Cmp(zeroBI) == 0) {
			continue
		}

		strI := strconv.Itoa(i)
		bin := Bin{
			ReserveA:  new(big.Int).Set(binRaw.BinState.ReserveA),
			ReserveB:  new(big.Int).Set(binRaw.BinState.ReserveB),
			LowerTick: big.NewInt(int64(binRaw.BinState.LowerTick)),
			Kind:      big.NewInt(int64(binRaw.BinState.Kind)),
			MergeID:   new(big.Int).Set(binRaw.BinState.MergeID),
		}
		bins[strI] = bin

		if bin.MergeID.Int64() == 0 {
			binIndex := d.putTypeAtTick(binMap, binMapHex, bin.Kind, bin.LowerTick)
			if binPositions[bin.LowerTick.String()] == nil {
				binPositions[bin.LowerTick.String()] = make(map[string]*big.Int)
			}
			binPositions[bin.LowerTick.String()][bin.Kind.String()] = big.NewInt(int64(i))

			if minBinMapIndex == nil {
				minBinMapIndex = new(big.Int).Set(binIndex)
			} else if minBinMapIndex.Cmp(binIndex) > 0 {
				minBinMapIndex.Set(binIndex)
			}
			if maxBinMapIndex == nil {
				maxBinMapIndex = new(big.Int).Set(binIndex)
			} else if maxBinMapIndex.Cmp(binIndex) < 0 {
				maxBinMapIndex.Set(binIndex)
			}
		}
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("faield to unmarshal static extra")

		return entity.Pool{}, err
	}
	_, _, sqrtPrice, liquidity, _, _ := currentTickLiquidity(activeTick, &MaverickPoolState{
		TickSpacing:      staticExtra.TickSpacing,
		Fee:              fee,
		ProtocolFeeRatio: protocolFeeRatio,
		ActiveTick:       activeTick,
		BinCounter:       binCounter,
		Bins:             bins,
		BinPositions:     binPositions,
		BinMap:           binMap,
		BinMapHex:        binMapHex,
		minBinMapIndex:   minBinMapIndex,
		maxBinMapIndex:   maxBinMapIndex,
	})

	var extra = Extra{
		Fee:              fee,
		ProtocolFeeRatio: protocolFeeRatio,
		ActiveTick:       activeTick,
		BinCounter:       binCounter,
		Bins:             bins,
		BinPositions:     binPositions,
		BinMap:           binMap,
		BinMapHex:        binMapHex,

		SqrtPriceX96: sqrtPrice,
		Liquidity:    liquidity,

		MinBinMapIndex: minBinMapIndex,
		MaxBinMapIndex: maxBinMapIndex,
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

func (d *PoolTracker) putTypeAtTick(
	binMap map[string]*big.Int,
	binMapHex map[string]*big.Int,
	kind, tick *big.Int,
) *big.Int {
	offset, mapIndex := d.getMapPointer(
		new(big.Int).Add(
			new(big.Int).Mul(tick, Kinds),
			kind,
		))
	subMap := binMap[mapIndex.String()]
	if subMap == nil {
		subMap = big.NewInt(0)
	}

	value := new(big.Int).Or(
		subMap,
		new(big.Int).Lsh(big.NewInt(1), uint(offset.Int64())))

	binMap[mapIndex.String()] = value
	binMapHex[mapIndex.Text(16)] = value
	return mapIndex
}

func (d *PoolTracker) getMapPointer(tick *big.Int) (*big.Int, *big.Int) {
	offset := new(big.Int).And(tick, OffsetMask)
	mapIndex := new(big.Int).Rsh(tick, 8)

	return offset, mapIndex
}
