package dexLite

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       config,
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
	// Extract dexId from existing pool extra data
	var existingExtra PoolExtra
	if err := json.Unmarshal([]byte(p.Extra), &existingExtra); err != nil {
		return p, err
	}

	poolState, blockNumber, blockTimestamp, err := t.getPoolStateByDexId(ctx, existingExtra.DexId, overrides)
	if err != nil {
		return p, err
	}

	// Update pool state while keeping dexKey and dexId
	extra := PoolExtra{
		DexKey:         existingExtra.DexKey,
		DexId:          existingExtra.DexId,
		PoolState:      *poolState,
		BlockTimestamp: blockTimestamp,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("Error marshaling extra data")
		return p, err
	}

	// Update reserves and fee for KyberSwap routing
	reserves, fee := t.calculatePoolMetrics(poolState, p.Tokens)
	p.Reserves = reserves
	p.SwapFee = fee

	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()

	return p, nil
}

// calculatePoolMetrics computes real reserves and fee for KyberSwap routing
func (t *PoolTracker) calculatePoolMetrics(poolState *PoolState, tokens []*entity.PoolToken) (entity.PoolReserves, float64) {
	// Extract fee from dexVariables (bits 0-12, stored as basis points)
	feeRaw := new(big.Int).And(poolState.DexVariables, X13)
	fee := float64(feeRaw.Int64()) / FeePercentPrecision

	// Extract total supplies and calculate reserves
	// Unpack dexVariables to get token supplies in internal precision (9 decimals)
	unpackedVars := t.unpackDexVariables(poolState.DexVariables)

	// Convert internal supplies to actual token decimals
	token0Supply := t.adjustFromInternalDecimals(unpackedVars.Token0TotalSupplyAdjusted, tokens[0].Decimals)
	token1Supply := t.adjustFromInternalDecimals(unpackedVars.Token1TotalSupplyAdjusted, tokens[1].Decimals)

	reserves := entity.PoolReserves{
		token0Supply.String(),
		token1Supply.String(),
	}

	return reserves, fee
}

// unpackDexVariables extracts the packed variables from dexVariables
func (t *PoolTracker) unpackDexVariables(dexVariables *big.Int) UnpackedDexVariables {
	return UnpackedDexVariables{
		Fee:                         new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesFee), X13),
		RevenueCut:                  new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesRevenueCut), X7),
		RebalancingStatus:           new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesRebalancingStatus), X2),
		CenterPriceShiftActive:      new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesCenterPriceShiftActive), X1).Cmp(big.NewInt(1)) == 0,
		CenterPrice:                 new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesCenterPrice), X40),
		CenterPriceContractAddress:  new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesCenterPriceContractAddress), X19),
		RangePercentShiftActive:     new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesRangePercentShiftActive), X1).Cmp(big.NewInt(1)) == 0,
		UpperPercent:                new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesUpperPercent), X14),
		LowerPercent:                new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesLowerPercent), X14),
		ThresholdPercentShiftActive: new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesThresholdPercentShiftActive), X1).Cmp(big.NewInt(1)) == 0,
		UpperShiftThresholdPercent:  new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesUpperShiftThresholdPercent), X7),
		LowerShiftThresholdPercent:  new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesLowerShiftThresholdPercent), X7),
		Token0Decimals:              new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesToken0Decimals), X5),
		Token1Decimals:              new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesToken1Decimals), X5),
		Token0TotalSupplyAdjusted:   new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesToken0TotalSupplyAdjusted), X60),
		Token1TotalSupplyAdjusted:   new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesToken1TotalSupplyAdjusted), X60),
	}
}

// adjustFromInternalDecimals converts from 9-decimal precision to token decimals
func (t *PoolTracker) adjustFromInternalDecimals(amount *big.Int, tokenDecimals uint8) *big.Int {
	internalDecimals := uint8(9)
	if tokenDecimals >= internalDecimals {
		factor := tenPow(int(tokenDecimals - internalDecimals))
		return new(big.Int).Mul(amount, factor)
	} else {
		factor := tenPow(int(internalDecimals - tokenDecimals))
		return new(big.Int).Div(amount, factor)
	}
}

func (t *PoolTracker) getPoolStateByDexId(
	ctx context.Context,
	dexId [8]byte,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*PoolState, uint64, uint64, error) {

	var poolStateSlots [4]*big.Int

	req := t.ethrpcClient.R().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	// Read the 4 storage variables for FluidDexLite pool state
	for i := 0; i < 4; i++ {
		slot := t.calculatePoolStateSlot(dexId, i)
		poolStateSlots[i] = new(big.Int)
		req.AddCall(&ethrpc.Call{
			ABI:    fluidDexLiteABI,
			Target: t.config.DexLiteAddress,
			Method: SRMethodReadFromStorage,
			Params: []interface{}{slot},
		}, []interface{}{poolStateSlots[i]})
	}

	resp, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("Failed to get pool state")
		return nil, 0, 0, err
	}

	// Get block timestamp
	blockTimestamp, err := t.ethrpcClient.R().SetContext(ctx).GetCurrentBlockTimestamp()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("Failed to get block timestamp")
		return nil, 0, 0, err
	}

	// Just return the 4 state variables - PoolSimulator will handle pause logic
	poolState := &PoolState{
		DexVariables:     poolStateSlots[0],
		CenterPriceShift: poolStateSlots[1],
		RangeShift:       poolStateSlots[2],
		ThresholdShift:   poolStateSlots[3],
	}

	return poolState, resp.BlockNumber.Uint64(), blockTimestamp, nil
}

// Helper functions
func (t *PoolTracker) calculateDexId(dexKey DexKey) [8]byte {
	// dexId = bytes8(keccak256(abi.encode(dexKey)))
	// Encode the DexKey similar to abi.encode
	data := make([]byte, 0, 96) // 32 + 32 + 32 bytes
	data = append(data, common.LeftPadBytes(dexKey.Token0.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(dexKey.Token1.Bytes(), 32)...)
	data = append(data, dexKey.Salt[:]...)

	hash := crypto.Keccak256(data)

	var dexId [8]byte
	copy(dexId[:], hash[:8])
	return dexId
}

func (t *PoolTracker) calculatePoolStateSlot(dexId [8]byte, offset int) common.Hash {
	// Pool state mapping: _dexVariables is at slot 2, others follow
	// keccak256(dexId, baseSlot) where baseSlot varies by field
	var baseSlot *big.Int

	switch offset {
	case 0: // _dexVariables mapping at slot 2
		baseSlot = big.NewInt(2)
	case 1: // _centerPriceShift mapping at slot 3
		baseSlot = big.NewInt(3)
	case 2: // _rangeShift mapping at slot 4
		baseSlot = big.NewInt(4)
	case 3: // _thresholdShift mapping at slot 5
		baseSlot = big.NewInt(5)
	default:
		baseSlot = big.NewInt(2)
	}

	data := make([]byte, 0, 40) // 8 + 32 bytes
	data = append(data, common.LeftPadBytes(dexId[:], 32)...)
	data = append(data, common.LeftPadBytes(baseSlot.Bytes(), 32)...)
	baseHash := crypto.Keccak256(data)

	return common.BytesToHash(baseHash)
}
