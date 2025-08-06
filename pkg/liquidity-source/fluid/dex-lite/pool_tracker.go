package dexLite

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
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

	poolState, blockNumber, err := t.getPoolStateByDexId(ctx, existingExtra.DexId, overrides)
	if err != nil {
		return p, err
	}

	// Update pool state while keeping dexKey and dexId
	extra := PoolExtra{
		DexKey:         existingExtra.DexKey,
		DexId:          existingExtra.DexId,
		PoolState:      *poolState,
		BlockTimestamp: uint64(time.Now().Unix()),
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("Error marshaling extra data")
		return p, err
	}

	// Update reserves and fee for KyberSwap routing
	reserves, fee := calculatePoolMetrics(poolState, p.Tokens)
	p.Reserves = reserves
	p.SwapFee = fee

	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()

	return p, nil
}

// calculatePoolMetrics computes real reserves and fee for KyberSwap routing
func calculatePoolMetrics(poolState *PoolState, tokens []*entity.PoolToken) (entity.PoolReserves, float64) {
	// Extract fee from dexVariables (bits 0-12, stored as basis points)
	feeRaw := new(uint256.Int).And(poolState.DexVariables, X13)
	fee := feeRaw.Float64() / FeePercentPrecision

	// Extract total supplies and calculate reserves
	// Unpack dexVariables to get token supplies in internal precision (9 decimals)
	unpackedVars := unpackDexVariables(poolState.DexVariables)

	// Convert internal supplies to actual token decimals
	token0Supply := adjustFromInternalDecimals(unpackedVars.Token0TotalSupplyAdjusted, tokens[0].Decimals)
	token1Supply := adjustFromInternalDecimals(unpackedVars.Token1TotalSupplyAdjusted, tokens[1].Decimals)

	reserves := entity.PoolReserves{
		token0Supply.String(),
		token1Supply.String(),
	}

	return reserves, fee
}

// unpackDexVariables extracts the packed variables from dexVariables
func unpackDexVariables(dexVars *uint256.Int) *UnpackedDexVariables {
	return &UnpackedDexVariables{
		Fee:                         rshAnd(dexVars, BitPosFee, X13),
		RevenueCut:                  rshAnd(dexVars, BitPosRevenueCut, X7),
		RebalancingStatus:           rshAnd(dexVars, BitPosRebalancingStatus, X2),
		CenterPriceShiftActive:      rshAnd(dexVars, BitPosCenterPriceShiftActive, X1).Cmp(big256.U1) == 0,
		CenterPrice:                 rshAnd(dexVars, BitPosCenterPrice, X40),
		CenterPriceContractAddress:  rshAnd(dexVars, BitPosCenterPriceContractAddress, X19),
		RangePercentShiftActive:     rshAnd(dexVars, BitPosRangePercentShiftActive, X1).Cmp(big256.U1) == 0,
		UpperPercent:                rshAnd(dexVars, BitPosUpperPercent, X14),
		LowerPercent:                rshAnd(dexVars, BitPosLowerPercent, X14),
		ThresholdPercentShiftActive: rshAnd(dexVars, BitPosThresholdPercentShiftActive, X1).Cmp(big256.U1) == 0,
		UpperShiftThresholdPercent:  rshAnd(dexVars, BitPosUpperShiftThresholdPercent, X7),
		LowerShiftThresholdPercent:  rshAnd(dexVars, BitPosLowerShiftThresholdPercent, X7),
		Token0TotalSupplyAdjusted:   rshAnd(dexVars, BitPosToken0TotalSupplyAdjusted, X60),
		Token1TotalSupplyAdjusted:   rshAnd(dexVars, BitPosToken1TotalSupplyAdjusted, X60),
	}
}

func unpackTotalSupplies(dexVariables *uint256.Int) (*uint256.Int, *uint256.Int) {
	return rshAnd(dexVariables, BitPosToken0TotalSupplyAdjusted, X60),
		rshAnd(dexVariables, BitPosToken1TotalSupplyAdjusted, X60)
}

func rshAnd(value *uint256.Int, shift uint, mask *uint256.Int) *uint256.Int {
	var ret uint256.Int
	return ret.And(ret.Rsh(value, shift), mask)
}

// adjustFromInternalDecimals converts from internal decimal precision to token decimals
func adjustFromInternalDecimals(amount *uint256.Int, tokenDecimals uint8) *uint256.Int {
	if tokenDecimals >= TokensDecimalsPrecision {
		factor := big256.TenPow(tokenDecimals - TokensDecimalsPrecision)
		return new(uint256.Int).Mul(amount, factor)
	} else {
		factor := big256.TenPow(TokensDecimalsPrecision - tokenDecimals)
		return new(uint256.Int).Div(amount, factor)
	}
}

func (t *PoolTracker) getPoolStateByDexId(
	ctx context.Context,
	dexId [8]byte,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*PoolState, uint64, error) {

	// NOTE: Direct ethClient calls don't support overrides, so we log a warning if provided
	if overrides != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
		}).Warn("Overrides not supported with direct ethClient calls")
	}

	var poolStateSlots [4]*big.Int

	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetRequireSuccess(true).SetOverrides(overrides)
	for i := range 4 {
		slot := t.calculatePoolStateSlot(dexId, i)
		req.AddCall(&ethrpc.Call{
			ABI:    fluidDexLiteABI,
			Target: t.config.DexLiteAddress,
			Method: "readFromStorage",
			Params: []any{slot},
		}, []any{&poolStateSlots[i]})
	}
	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("Failed to read pool state slot")
		return nil, 0, err
	}

	// Just return the 4 state variables - PoolSimulator will handle pause logic
	poolState := &PoolState{
		DexVariables:     uint256.MustFromBig(poolStateSlots[0]),
		CenterPriceShift: uint256.MustFromBig(poolStateSlots[1]),
		RangeShift:       uint256.MustFromBig(poolStateSlots[2]),
		ThresholdShift:   uint256.MustFromBig(poolStateSlots[3]),
	}

	return poolState, resp.BlockNumber.Uint64(), nil
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
	// Storage slot mapping for FluidDexLite pool state variables
	var baseSlot uint64
	switch offset {
	case 0: // _dexVariables mapping
		baseSlot = StorageSlotDexVariables
	case 1: // _centerPriceShift mapping
		baseSlot = StorageSlotCenterPriceShift
	case 2: // _rangeShift mapping
		baseSlot = StorageSlotRangeShift
	case 3: // _thresholdShift mapping
		baseSlot = StorageSlotThresholdShift
	default:
		baseSlot = StorageSlotDexVariables
	}

	// Convert bytes8 dexId to bytes32 (right-padded with zeros)
	var dexIdBytes32 common.Hash
	copy(dexIdBytes32[:], dexId[:])

	// Use Solidity mapping storage calculation: keccak256(abi.encode(key, slot))
	uint256Type, _ := abi.NewType("uint256", "", nil)
	bytes32Type, _ := abi.NewType("bytes32", "", nil)
	arguments := abi.Arguments{{Type: bytes32Type}, {Type: uint256Type}}

	var tmp big.Int
	encoded, err := arguments.Pack(common.BytesToHash(dexIdBytes32[:]), tmp.SetUint64(baseSlot))
	if err != nil {
		// Fallback to manual encoding
		encoded = make([]byte, 64)
		copy(encoded[:32], dexIdBytes32[:])
		copy(encoded[32:], common.LeftPadBytes(tmp.SetUint64(baseSlot).Bytes(), 32))
	}

	return common.BytesToHash(crypto.Keccak256(encoded))
}
