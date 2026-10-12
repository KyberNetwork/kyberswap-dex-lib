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
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

const (
	depositXIndex = 1
	depositYIndex = 2
	borrowXIndex  = 4
	borrowYIndex  = 5

	defaultRefreshInterval = 5 * time.Minute
)

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

type reservesRPC struct {
	ReserveX            *big.Int
	ReserveY            *big.Int
	LastUpdateTimestamp uint32
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
	Beacon         common.Address
	Implementation common.Address
	Reserves       reservesRPC
	References     referenceReservesRPC
	Totals         totalAssetsAndSharesRPC
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
	stateResponse, err := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber).
		AddCall(&ethrpc.Call{
			ABI: pairABI, Target: p.Address, Method: "getReserves",
		}, []any{&snapshot.Reserves}).
		AddCall(&ethrpc.Call{
			ABI: pairABI, Target: p.Address, Method: "referenceReserves",
		}, []any{&snapshot.References}).
		AddCall(&ethrpc.Call{
			ABI: pairABI, Target: p.Address, Method: "totalAssetsAndShares", Params: []any{true},
		}, []any{&snapshot.Totals}).
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

	return updatePoolFromSnapshot(p, t.config, staticExtra, snapshot, time.Now().Unix(), len(params.Logs) > 0)
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
	hasLogs bool,
) (entity.Pool, error) {
	if p.BlockNumber > snapshot.BlockNumber {
		return p, nil
	}

	staticExtra, err := reconcileTrackerMetadata(config, staticExtra, snapshot.Beacon, snapshot.Implementation)
	if err != nil {
		return p, err
	}

	values := []*big.Int{
		snapshot.Reserves.ReserveX,
		snapshot.Reserves.ReserveY,
		snapshot.References.ReferenceReserveX,
		snapshot.References.ReferenceReserveY,
		snapshot.Totals.AllAssets[depositXIndex],
		snapshot.Totals.AllAssets[depositYIndex],
		snapshot.Totals.AllAssets[borrowXIndex],
		snapshot.Totals.AllAssets[borrowYIndex],
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

	missingX := positiveDifference(snapshot.Totals.AllAssets[borrowXIndex], snapshot.Totals.AllAssets[depositXIndex])
	missingY := positiveDifference(snapshot.Totals.AllAssets[borrowYIndex], snapshot.Totals.AllAssets[depositYIndex])
	if missingX.Cmp(snapshot.Reserves.ReserveX) >= 0 || missingY.Cmp(snapshot.Reserves.ReserveY) >= 0 {
		return p, ErrInvalidPool
	}

	extra := Extra{
		LastUpdateTimestamp:  snapshot.Reserves.LastUpdateTimestamp,
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

	p.Reserves = entity.PoolReserves{snapshot.Reserves.ReserveX.String(), snapshot.Reserves.ReserveY.String()}
	p.Extra = string(extraBytes)
	p.StaticExtra = string(staticExtraBytes)
	p.BlockNumber = snapshot.BlockNumber
	if hasLogs {
		p.Timestamp = now
	}
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
