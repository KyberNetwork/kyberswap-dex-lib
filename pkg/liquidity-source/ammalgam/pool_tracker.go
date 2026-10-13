package ammalgam

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	utileth "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

const (
	depositXIndex = 1
	depositYIndex = 2
	borrowXIndex  = 4
	borrowYIndex  = 5

	defaultRefreshInterval = 5 * time.Minute
	interestPeriodForSwap  = uint32(24*time.Hour/time.Second) - 1
)

var (
	// AmmalgamPair v1.1.0 packs raw reserves and timestamps in these slots.
	// The configured beacon implementation is checked before the decoded state
	// is accepted, so an implementation upgrade fails closed.
	pairReservesSlot   = common.BigToHash(big.NewInt(16))
	pairReferencesSlot = common.BigToHash(big.NewInt(17))
	uint112Mask        = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 112), big.NewInt(1))
)

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

type pairStorage struct {
	ReserveX             *big.Int
	ReserveY             *big.Int
	LastUpdateTimestamp  uint32
	ReferenceReserveX    *big.Int
	ReferenceReserveY    *big.Int
	LastLendingTimestamp uint32
}

type referenceReservesRPC struct {
	ReferenceReserveX *big.Int
	ReferenceReserveY *big.Int
}

type totalAssetsAndSharesRPC struct {
	AllAssets [6]*big.Int
	AllShares [6]*big.Int
}

type trackerSnapshot struct {
	BlockNumber    uint64
	BlockTimestamp uint32
	Beacon         common.Address
	Implementation common.Address
	Storage        pairStorage
	References     referenceReservesRPC
	RawTotals      totalAssetsAndSharesRPC
	AccruedTotals  totalAssetsAndSharesRPC
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	if config == nil || !validPairAddress(config.PairImplementation) {
		return nil, errors.New("Ammalgam pair implementation is required for the v1.1.0 storage layout")
	}
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	if t == nil || t.config == nil || t.ethrpcClient == nil ||
		!validPairAddress(t.config.FactoryAddress) || !validPairAddress(p.Address) {
		return p, errors.New("invalid Ammalgam tracker configuration")
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, fmt.Errorf("decode Ammalgam static extra: %w", err)
	}

	var beacon common.Address
	beaconResponse, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI: factoryABI, Target: t.config.FactoryAddress, Method: "pairBeacon",
		}, []any{&beacon}).
		TryBlockAndAggregate()
	if err != nil {
		return p, fmt.Errorf("read Ammalgam pair beacon: %w", err)
	}
	if err := requireAllCallsSucceeded(beaconResponse, 1); err != nil {
		return p, err
	}
	if beaconResponse.BlockNumber == nil || !beaconResponse.BlockNumber.IsUint64() ||
		!validPairAddress(beacon.Hex()) {
		return p, errors.New("invalid Ammalgam pair beacon response")
	}

	blockNumber := new(big.Int).Set(beaconResponse.BlockNumber)
	snapshot := trackerSnapshot{BlockNumber: blockNumber.Uint64(), Beacon: beacon}
	header, err := t.ethrpcClient.GetETHClient().HeaderByNumber(ctx, blockNumber)
	if err != nil {
		return p, fmt.Errorf("read Ammalgam snapshot header: %w", err)
	}
	if header == nil || header.Number == nil || header.Number.Cmp(blockNumber) != 0 {
		return p, errors.New("invalid Ammalgam snapshot header")
	}
	snapshot.BlockTimestamp = uint32(header.Time)

	storageReads := []utileth.StorageRead{{Slot: pairReservesSlot}, {Slot: pairReferencesSlot}}
	if err := utileth.BatchGetStorageAtAtBlock(ctx, t.ethrpcClient.GetETHClient().Client(),
		common.HexToAddress(p.Address), storageReads, blockNumber, utileth.DefaultBatchRetry); err != nil {
		return p, fmt.Errorf("read Ammalgam pair storage: %w", err)
	}
	snapshot.Storage = decodePairStorageV110(storageReads[0].Result, storageReads[1].Result)

	stateResponse, err := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber).
		AddCall(&ethrpc.Call{
			ABI: pairABI, Target: p.Address, Method: "referenceReserves",
		}, []any{&snapshot.References}).
		AddCall(&ethrpc.Call{
			ABI: pairABI, Target: p.Address, Method: "totalAssetsAndShares", Params: []any{false},
		}, []any{&snapshot.RawTotals}).
		AddCall(&ethrpc.Call{
			ABI: pairABI, Target: p.Address, Method: "totalAssetsAndShares", Params: []any{true},
		}, []any{&snapshot.AccruedTotals}).
		AddCall(&ethrpc.Call{
			ABI: beaconABI, Target: beacon.Hex(), Method: "implementation",
		}, []any{&snapshot.Implementation}).
		TryAggregate()
	if err != nil {
		return p, fmt.Errorf("read Ammalgam pool state: %w", err)
	}
	if err := requireAllCallsSucceeded(stateResponse, 4); err != nil {
		return p, err
	}

	return updatePoolFromSnapshot(p, t.config, staticExtra, snapshot, time.Now().Unix(), activityTimestamp(p, params))
}

func activityTimestamp(p entity.Pool, params pool.GetNewPoolStateParams) int64 {
	if len(params.Logs) == 0 || params.BlockHeaders == nil {
		return p.Timestamp
	}
	latest := params.Logs[len(params.Logs)-1]
	if isZeroLog(latest) {
		return p.Timestamp
	}
	if header, ok := params.BlockHeaders[latest.BlockNumber]; ok {
		return max(p.Timestamp, int64(header.Timestamp))
	}
	return p.Timestamp
}

func isZeroLog(log ethtypes.Log) bool {
	return log.Address == (common.Address{}) && log.BlockNumber == 0 && log.BlockHash == (common.Hash{}) &&
		log.TxHash == (common.Hash{}) && len(log.Topics) == 0 && len(log.Data) == 0
}

func decodePairStorageV110(reservesWord, referencesWord common.Hash) pairStorage {
	decode112 := func(word *big.Int, shift uint) *big.Int {
		return new(big.Int).And(new(big.Int).Rsh(new(big.Int).Set(word), shift), uint112Mask)
	}
	reserves := reservesWord.Big()
	references := referencesWord.Big()
	return pairStorage{
		ReserveX:             decode112(reserves, 0),
		ReserveY:             decode112(reserves, 112),
		LastUpdateTimestamp:  uint32(new(big.Int).Rsh(new(big.Int).Set(reserves), 224).Uint64()),
		ReferenceReserveX:    decode112(references, 0),
		ReferenceReserveY:    decode112(references, 112),
		LastLendingTimestamp: uint32(new(big.Int).Rsh(new(big.Int).Set(references), 224).Uint64()),
	}
}

func resolveSwapState(snapshot trackerSnapshot) ([2]*big.Int, totalAssetsAndSharesRPC, error) {
	reserves := [2]*big.Int{snapshot.Storage.ReserveX, snapshot.Storage.ReserveY}
	if reserves[0] == nil || reserves[1] == nil {
		return reserves, totalAssetsAndSharesRPC{}, ErrInvalidPool
	}
	selected := snapshot.RawTotals
	// Solidity intentionally uses uint32 timestamp subtraction so the clock
	// continues to work across its wrap boundary.
	if snapshot.BlockTimestamp-snapshot.Storage.LastLendingTimestamp < interestPeriodForSwap {
		return [2]*big.Int{new(big.Int).Set(reserves[0]), new(big.Int).Set(reserves[1])}, selected, nil
	}

	selected = snapshot.AccruedTotals
	for i, indices := range [2][2]int{{depositXIndex, borrowXIndex}, {depositYIndex, borrowYIndex}} {
		depositIndex, borrowIndex := indices[0], indices[1]
		rawDeposit, rawBorrow := snapshot.RawTotals.AllAssets[depositIndex], snapshot.RawTotals.AllAssets[borrowIndex]
		accruedDeposit := snapshot.AccruedTotals.AllAssets[depositIndex]
		accruedBorrow := snapshot.AccruedTotals.AllAssets[borrowIndex]
		if rawDeposit == nil || rawBorrow == nil || accruedDeposit == nil || accruedBorrow == nil ||
			accruedDeposit.Cmp(rawDeposit) < 0 || accruedBorrow.Cmp(rawBorrow) < 0 {
			return reserves, totalAssetsAndSharesRPC{}, ErrInvalidPool
		}
		depositGrowth := new(big.Int).Sub(accruedDeposit, rawDeposit)
		borrowGrowth := new(big.Int).Sub(accruedBorrow, rawBorrow)
		if borrowGrowth.Cmp(depositGrowth) < 0 {
			return reserves, totalAssetsAndSharesRPC{}, ErrInvalidPool
		}
		// totalAssetsAndShares(true) adds protocol fees to both the deposit
		// and borrow totals, while the LP portion is withheld from deposits:
		// LP interest = borrow growth - deposit growth.
		lpInterest := new(big.Int).Sub(borrowGrowth, depositGrowth)
		reserves[i] = new(big.Int).Add(reserves[i], lpInterest)
	}
	return reserves, selected, nil
}

func (t *PoolTracker) NextUpdateAt(p entity.Pool) int64 {
	refreshInterval := defaultRefreshInterval
	if t != nil && t.config != nil && t.config.RefreshInterval > 0 {
		refreshInterval = t.config.RefreshInterval
	}
	seconds := int64(refreshInterval / time.Second)
	if seconds <= 0 {
		seconds = int64(defaultRefreshInterval / time.Second)
	}
	lastRefresh := p.Timestamp
	var extra Extra
	if json.Unmarshal([]byte(p.Extra), &extra) == nil && extra.LastRefreshTimestamp > 0 {
		lastRefresh = extra.LastRefreshTimestamp
	}
	return lastRefresh + seconds
}

func reconcileTrackerMetadata(
	config *Config,
	staticExtra StaticExtra,
	beacon common.Address,
	implementation common.Address,
) (StaticExtra, error) {
	if config == nil || !validPairAddress(config.FactoryAddress) ||
		!validPairAddress(beacon.Hex()) || !validPairAddress(implementation.Hex()) {
		return staticExtra, errors.New("invalid Ammalgam tracker metadata")
	}
	if staticExtra.FactoryAddress != "" &&
		(!validPairAddress(staticExtra.FactoryAddress) ||
			!strings.EqualFold(staticExtra.FactoryAddress, config.FactoryAddress)) {
		return staticExtra, fmt.Errorf("Ammalgam factory mismatch: configured %s, stored %s",
			config.FactoryAddress, staticExtra.FactoryAddress)
	}
	if config.PairBeacon != "" &&
		(!validPairAddress(config.PairBeacon) || !strings.EqualFold(config.PairBeacon, beacon.Hex())) {
		return staticExtra, fmt.Errorf("Ammalgam pair beacon mismatch: configured %s, on chain %s",
			config.PairBeacon, beacon.Hex())
	}
	if staticExtra.PairBeacon != "" &&
		(!validPairAddress(staticExtra.PairBeacon) || !strings.EqualFold(staticExtra.PairBeacon, beacon.Hex())) {
		return staticExtra, fmt.Errorf("Ammalgam pair beacon mismatch: stored %s, on chain %s",
			staticExtra.PairBeacon, beacon.Hex())
	}
	if config.PairImplementation != "" &&
		(!validPairAddress(config.PairImplementation) ||
			!strings.EqualFold(config.PairImplementation, implementation.Hex())) {
		return staticExtra, fmt.Errorf("Ammalgam pair implementation mismatch: configured %s, on chain %s",
			config.PairImplementation, implementation.Hex())
	}
	if staticExtra.PairImplementation != "" &&
		(!validPairAddress(staticExtra.PairImplementation) ||
			!strings.EqualFold(staticExtra.PairImplementation, implementation.Hex())) {
		return staticExtra, fmt.Errorf("Ammalgam pair implementation mismatch: stored %s, on chain %s",
			staticExtra.PairImplementation, implementation.Hex())
	}

	factory := common.HexToAddress(config.FactoryAddress)
	staticExtra.FactoryAddress = hexutil.Encode(factory[:])
	staticExtra.PairBeacon = hexutil.Encode(beacon[:])
	staticExtra.PairImplementation = hexutil.Encode(implementation[:])
	return staticExtra, nil
}

func updatePoolFromSnapshot(
	p entity.Pool,
	config *Config,
	staticExtra StaticExtra,
	snapshot trackerSnapshot,
	now int64,
	lastActivityTimestamp int64,
) (entity.Pool, error) {
	if p.BlockNumber > snapshot.BlockNumber {
		return p, nil
	}

	staticExtra, err := reconcileTrackerMetadata(config, staticExtra, snapshot.Beacon, snapshot.Implementation)
	if err != nil {
		return p, err
	}

	reserves, totals, err := resolveSwapState(snapshot)
	if err != nil {
		return p, err
	}
	values := []*big.Int{
		reserves[0],
		reserves[1],
		snapshot.References.ReferenceReserveX,
		snapshot.References.ReferenceReserveY,
		totals.AllAssets[depositXIndex],
		totals.AllAssets[depositYIndex],
		totals.AllAssets[borrowXIndex],
		totals.AllAssets[borrowYIndex],
	}
	for _, value := range values {
		if !isUint112(value) {
			return p, ErrReserveLimit
		}
	}
	if snapshot.References.ReferenceReserveX.Sign() == 0 ||
		snapshot.References.ReferenceReserveY.Sign() == 0 {
		return p, ErrInvalidPool
	}

	missingX := positiveDifference(totals.AllAssets[borrowXIndex], totals.AllAssets[depositXIndex])
	missingY := positiveDifference(totals.AllAssets[borrowYIndex], totals.AllAssets[depositYIndex])
	if missingX.Cmp(reserves[0]) >= 0 || missingY.Cmp(reserves[1]) >= 0 {
		return p, ErrInvalidPool
	}

	extra := Extra{
		LastUpdateTimestamp:  snapshot.Storage.LastUpdateTimestamp,
		LastRefreshTimestamp: now,
	}
	extra.ReferenceReserves[0] = *uint256.MustFromBig(snapshot.References.ReferenceReserveX)
	extra.ReferenceReserves[1] = *uint256.MustFromBig(snapshot.References.ReferenceReserveY)
	extra.MissingAssets[0] = *uint256.MustFromBig(missingX)
	extra.MissingAssets[1] = *uint256.MustFromBig(missingY)
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, fmt.Errorf("marshal Ammalgam extra: %w", err)
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return p, fmt.Errorf("marshal Ammalgam static extra: %w", err)
	}

	p.Reserves = entity.PoolReserves{reserves[0].String(), reserves[1].String()}
	p.Extra = string(extraBytes)
	p.StaticExtra = string(staticExtraBytes)
	p.BlockNumber = snapshot.BlockNumber
	p.Timestamp = max(p.Timestamp, lastActivityTimestamp)
	return p, nil
}

func isUint112(value *big.Int) bool {
	return value != nil && value.Sign() >= 0 && value.BitLen() <= 112
}

func positiveDifference(x, y *big.Int) *big.Int {
	if x.Cmp(y) <= 0 {
		return new(big.Int)
	}
	return new(big.Int).Sub(x, y)
}
