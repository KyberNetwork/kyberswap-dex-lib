package maverickv1

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexTypeMaverickV1, NewPoolTracker)

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
	activeTick := getStateResult.State.ActiveTick
	protocolFeeRatio := uint256.NewInt(getStateResult.State.ProtocolFeeRatio)

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
	bins := make(map[uint32]Bin)
	binPositions := make(map[int32]map[uint8]uint32)
	binMap := make(map[int16]*uint256.Int)
	for i, binRaw := range binRaws {
		i := uint32(i)
		if binRaw.BinState.MergeID.Sign() != 0 ||
			(binRaw.BinState.ReserveA.Sign() == 0 && binRaw.BinState.ReserveB.Sign() == 0) {
			continue
		}

		bin := Bin{
			ReserveA:  uint256.MustFromBig(binRaw.BinState.ReserveA),
			ReserveB:  uint256.MustFromBig(binRaw.BinState.ReserveB),
			LowerTick: binRaw.BinState.LowerTick,
			Kind:      binRaw.BinState.Kind,
		}
		bins[i] = bin

		if binRaw.BinState.MergeID.Sign() == 0 {
			_ = d.putTypeAtTick(binMap, bin.Kind, bin.LowerTick)
			binPosition := binPositions[bin.LowerTick]
			if binPosition == nil {
				binPosition = make(map[uint8]uint32)
				binPositions[bin.LowerTick] = binPosition
			}
			binPosition[bin.Kind] = i
		}
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to unmarshal static extra")

		return entity.Pool{}, err
	}
	feeU := uint256.MustFromBig(fee)
	_, _, sqrtPrice, liquidity, _, _ := currentTickLiquidity(activeTick, &MaverickPoolState{
		TickSpacing:      staticExtra.TickSpacing,
		Fee:              feeU,
		ProtocolFeeRatio: protocolFeeRatio,
		ActiveTick:       activeTick,
		Bins:             bins,
		BinPositions:     binPositions,
		BinMap:           binMap,
	})

	var extra = Extra{
		Fee:              feeU,
		ProtocolFeeRatio: protocolFeeRatio,
		ActiveTick:       activeTick,
		Bins:             bins,
		BinPositions:     binPositions,
		BinMap:           binMap,

		SqrtPriceX96: sqrtPrice,
		Liquidity:    liquidity,
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

func (d *PoolTracker) putTypeAtTick(binMap map[int16]*uint256.Int, kind uint8, tick int32) int16 {
	mapIndex, offset := getMapPointer(tick*Kinds + int32(kind))
	subMap := binMap[mapIndex]
	if subMap == nil {
		subMap = new(uint256.Int)
	}
	subMap[offset/64] |= 1 << (offset % 64)
	binMap[mapIndex] = subMap
	return mapIndex
}
