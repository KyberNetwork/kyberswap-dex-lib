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
)

const (
	depositXIndex         = 1
	depositYIndex         = 2
	borrowXIndex          = 4
	borrowYIndex          = 5
	interestPeriodForSwap = uint32(86_399)

	defaultRefreshInterval = 5 * time.Minute
)

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

type getReservesRPC struct {
	ReserveX             *big.Int
	ReserveY             *big.Int
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
	BlockTimestamp uint64
	Beacon         common.Address
	Reserves       getReservesRPC
	References     referenceReservesRPC
	RawTotals      totalAssetsAndSharesRPC
	AccruedTotals  totalAssetsAndSharesRPC
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
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
	snapshot.BlockTimestamp = header.Time

	stateResponse, err := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber).
		AddCall(&ethrpc.Call{
			ABI: pairABI, Target: p.Address, Method: "getReserves",
		}, []any{&snapshot.Reserves}).
		AddCall(&ethrpc.Call{
			ABI: pairABI, Target: p.Address, Method: "referenceReserves",
		}, []any{&snapshot.References}).
		AddCall(&ethrpc.Call{
			ABI: pairABI, Target: p.Address, Method: "totalAssetsAndShares", Params: []any{false},
		}, []any{&snapshot.RawTotals}).
		AddCall(&ethrpc.Call{
			ABI: pairABI, Target: p.Address, Method: "totalAssetsAndShares", Params: []any{true},
		}, []any{&snapshot.AccruedTotals}).
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

func resolveSwapState(snapshot trackerSnapshot) ([2]*big.Int, totalAssetsAndSharesRPC, error) {
	reserves := [2]*big.Int{snapshot.Reserves.ReserveX, snapshot.Reserves.ReserveY}
	if reserves[0] == nil || reserves[1] == nil {
		return reserves, totalAssetsAndSharesRPC{}, ErrInvalidPool
	}
	selected := snapshot.RawTotals
	elapsed := uint32(snapshot.BlockTimestamp) - snapshot.Reserves.LastLendingTimestamp
	if elapsed >= interestPeriodForSwap {
		selected = snapshot.AccruedTotals
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
) (StaticExtra, error) {
	if config == nil || !validPairAddress(config.FactoryAddress) ||
		!validPairAddress(beacon.Hex()) {
		return staticExtra, errors.New("invalid Ammalgam tracker metadata")
	}
	if staticExtra.FactoryAddress != "" &&
		(!validPairAddress(staticExtra.FactoryAddress) ||
			!strings.EqualFold(staticExtra.FactoryAddress, config.FactoryAddress)) {
		return staticExtra, fmt.Errorf("ammalgam factory mismatch: configured %s, stored %s",
			config.FactoryAddress, staticExtra.FactoryAddress)
	}
	if config.PairBeacon != "" &&
		(!validPairAddress(config.PairBeacon) || !strings.EqualFold(config.PairBeacon, beacon.Hex())) {
		return staticExtra, fmt.Errorf("ammalgam pair beacon mismatch: configured %s, on chain %s",
			config.PairBeacon, beacon.Hex())
	}
	if staticExtra.PairBeacon != "" &&
		(!validPairAddress(staticExtra.PairBeacon) || !strings.EqualFold(staticExtra.PairBeacon, beacon.Hex())) {
		return staticExtra, fmt.Errorf("ammalgam pair beacon mismatch: stored %s, on chain %s",
			staticExtra.PairBeacon, beacon.Hex())
	}

	factory := common.HexToAddress(config.FactoryAddress)
	staticExtra.FactoryAddress = hexutil.Encode(factory[:])
	staticExtra.PairBeacon = hexutil.Encode(beacon[:])
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

	staticExtra, err := reconcileTrackerMetadata(config, staticExtra, snapshot.Beacon)
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
		LastLendingTimestamp: snapshot.Reserves.LastLendingTimestamp,
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
