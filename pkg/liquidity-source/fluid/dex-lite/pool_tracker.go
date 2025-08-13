package dexLite

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	poolState, blockNumber, err := t.getPoolStateByDexId(ctx, staticExtra.DexId, overrides)
	if err != nil {
		return p, err
	}

	// Update pool state while keeping dexKey and dexId
	extra := PoolExtraMarshal{
		PoolState: PoolStateHex{
			DexVariables:     poolState.DexVariables.Hex(),
			CenterPriceShift: poolState.CenterPriceShift.Hex(),
			RangeShift:       poolState.RangeShift.Hex(),
			NewCenterPrice:   poolState.NewCenterPrice.Hex(),
		},
		BlockTimestamp: uint64(time.Now().Unix()),
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("error marshaling extra data")
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
	if unpackedVars == nil {
		return entity.PoolReserves{"0", "0"}, 0
	}

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
	if dexVars == nil || dexVars.IsZero() {
		return nil
	}
	return &UnpackedDexVariables{
		Fee:                         rshAnd(dexVars, BitPosFee, X13),
		RevenueCut:                  rshAnd(dexVars, BitPosRevenueCut, X7),
		RebalancingStatus:           rshAnd(dexVars, BitPosRebalancingStatus, X2).Uint64(),
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
	dexId DexId,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*PoolState, uint64, error) {
	var poolStateSlots [3]*big.Int

	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetRequireSuccess(true).SetOverrides(overrides)
	for i, baseSlot := range []common.Hash{
		StorageSlotDexVariables,
		StorageSlotCenterPriceShift,
		StorageSlotRangeShift,
	} {
		slot := calculatePoolStateSlot(dexId, baseSlot)
		req.AddCall(&ethrpc.Call{
			ABI:    fluidDexLiteABI,
			Target: t.config.DexLiteAddress,
			Method: SRMethodReadFromStorage,
			Params: []any{slot},
		}, []any{&poolStateSlots[i]})
	}
	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("failed to read pool state slot")
		return nil, 0, err
	}

	// Just return the 4 state variables - PoolSimulator will handle pause logic
	poolState := &PoolState{
		DexVariables:     uint256.MustFromBig(poolStateSlots[0]),
		CenterPriceShift: uint256.MustFromBig(poolStateSlots[1]),
		RangeShift:       uint256.MustFromBig(poolStateSlots[2]),
		NewCenterPrice:   big256.U0,
	}

	centerPriceContractAddress := rshAnd(poolState.DexVariables, BitPosCenterPriceContractAddress, X19)
	if !centerPriceContractAddress.IsZero() {
		centerPrice, err := t.getCenterPrice(ctx, centerPriceContractAddress.Uint64(), overrides)
		if err != nil {
			return poolState, resp.BlockNumber.Uint64(), nil
		}
		poolState.NewCenterPrice = centerPrice
	}

	return poolState, resp.BlockNumber.Uint64(), nil
}

func (t *PoolTracker) getCenterPrice(ctx context.Context, centerPriceContractAddressNonce uint64,
	overrides map[common.Address]gethclient.OverrideAccount) (*uint256.Int, error) {
	var expandedCenterPrice *big.Int
	centerPriceSource := crypto.CreateAddress(t.config.DeployerAddress, centerPriceContractAddressNonce)
	if _, err := t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).AddCall(&ethrpc.Call{
		ABI:    centerPriceABI,
		Target: hexutil.Encode(centerPriceSource[:]),
		Method: CenterPriceMethodCenterPrice,
	}, []any{&expandedCenterPrice}).Call(); err != nil {
		logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("failed to get center price")
		return nil, err
	}

	centerPrice, ok := uint256.FromBig(expandedCenterPrice)
	if !ok {
		return nil, ErrCenterPriceOverflow
	}

	return centerPrice, nil
}

func calculatePoolStateSlot(dexId DexId, baseSlot common.Hash) common.Hash {
	// Use Solidity mapping storage calculation: keccak256(abi.encode(key, slot)), where bytes8 key is right-padded
	return crypto.Keccak256Hash(dexId[:], bytes8Padding, baseSlot[:])
}
