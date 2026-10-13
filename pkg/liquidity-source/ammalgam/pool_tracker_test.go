package ammalgam

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestUpdatePoolFromSnapshotDerivesMissingAssets(t *testing.T) {
	factory := common.HexToAddress("0x1111111111111111111111111111111111111111")
	beacon := common.HexToAddress("0x2222222222222222222222222222222222222222")
	implementation := common.HexToAddress("0x3333333333333333333333333333333333333333")
	snapshot := trackerSnapshot{
		BlockNumber:    10,
		Beacon:         beacon,
		Implementation: implementation,
		BlockTimestamp: 1_000,
		Storage: pairStorage{
			ReserveX:             big.NewInt(1_000),
			ReserveY:             big.NewInt(2_000),
			LastUpdateTimestamp:  77,
			LastLendingTimestamp: 900,
		},
		References: referenceReservesRPC{
			ReferenceReserveX: big.NewInt(900),
			ReferenceReserveY: big.NewInt(1_800),
		},
		RawTotals: totalAssetsAndSharesRPC{
			AllAssets: [6]*big.Int{
				big.NewInt(0), big.NewInt(10), big.NewInt(8),
				big.NewInt(0), big.NewInt(40), big.NewInt(5),
			},
			AllShares: [6]*big.Int{
				big.NewInt(0), big.NewInt(0), big.NewInt(0),
				big.NewInt(0), big.NewInt(0), big.NewInt(0),
			},
		},
		AccruedTotals: totalAssetsAndSharesRPC{AllAssets: [6]*big.Int{
			big.NewInt(0), big.NewInt(10), big.NewInt(8),
			big.NewInt(0), big.NewInt(40), big.NewInt(5),
		}},
	}
	p := entity.Pool{BlockNumber: 9, Timestamp: 1, Reserves: entity.PoolReserves{"1", "2"}}
	got, err := updatePoolFromSnapshot(p, &Config{FactoryAddress: factory.Hex()}, StaticExtra{}, snapshot,
		1_700_000_000, 1_700_000_000)
	require.NoError(t, err)
	require.Equal(t, entity.PoolReserves{"1000", "2000"}, got.Reserves)
	require.Equal(t, uint64(10), got.BlockNumber)
	require.Equal(t, int64(1_700_000_000), got.Timestamp)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(got.Extra), &extra))
	require.Equal(t, "900", extra.ReferenceReserves[0].String())
	require.Equal(t, "1800", extra.ReferenceReserves[1].String())
	require.Equal(t, "30", extra.MissingAssets[0].String())
	require.True(t, extra.MissingAssets[1].IsZero())
	require.Equal(t, uint32(77), extra.LastUpdateTimestamp)
	require.Equal(t, int64(1_700_000_000), extra.LastRefreshTimestamp)

	var staticExtra StaticExtra
	require.NoError(t, json.Unmarshal([]byte(got.StaticExtra), &staticExtra))
	require.Equal(t, "0x1111111111111111111111111111111111111111", staticExtra.FactoryAddress)
	require.Equal(t, "0x2222222222222222222222222222222222222222", staticExtra.PairBeacon)
	require.Equal(t, "0x3333333333333333333333333333333333333333", staticExtra.PairImplementation)
}

func TestResolveSwapStateUsesAccruedTotalsAtAdapterReserveCheckpointBoundary(t *testing.T) {
	snapshot := trackerSnapshot{
		BlockTimestamp: 1_000 + interestPeriodForSwap,
		Storage: pairStorage{
			ReserveX: big.NewInt(1_000), ReserveY: big.NewInt(2_000),
			LastUpdateTimestamp: 1_000, LastLendingTimestamp: 1_000,
		},
		RawTotals: totalAssetsAndSharesRPC{AllAssets: [6]*big.Int{
			big.NewInt(0), big.NewInt(100), big.NewInt(200),
			big.NewInt(0), big.NewInt(200), big.NewInt(150),
		}},
		AccruedTotals: totalAssetsAndSharesRPC{AllAssets: [6]*big.Int{
			big.NewInt(0), big.NewInt(107), big.NewInt(203),
			big.NewInt(0), big.NewInt(212), big.NewInt(154),
		}},
	}

	reserves, totals, err := resolveSwapState(snapshot)
	require.NoError(t, err)
	require.Equal(t, int64(1_000), reserves[0].Int64())
	require.Equal(t, int64(2_000), reserves[1].Int64())
	require.Same(t, snapshot.AccruedTotals.AllAssets[borrowXIndex], totals.AllAssets[borrowXIndex])
}

func TestResolveSwapStateKeepsAdapterReservesWhenOnlyLendingCheckpointIsDue(t *testing.T) {
	snapshot := trackerSnapshot{
		BlockTimestamp: 1_000 + interestPeriodForSwap,
		Storage: pairStorage{
			ReserveX: big.NewInt(1_000), ReserveY: big.NewInt(2_000),
			LastUpdateTimestamp: 1_000 + interestPeriodForSwap - 1, LastLendingTimestamp: 1_000,
		},
		RawTotals: totalAssetsAndSharesRPC{AllAssets: [6]*big.Int{
			big.NewInt(0), big.NewInt(100), big.NewInt(200),
			big.NewInt(0), big.NewInt(200), big.NewInt(150),
		}},
		AccruedTotals: totalAssetsAndSharesRPC{AllAssets: [6]*big.Int{
			big.NewInt(0), big.NewInt(107), big.NewInt(203),
			big.NewInt(0), big.NewInt(212), big.NewInt(154),
		}},
	}

	reserves, totals, err := resolveSwapState(snapshot)
	require.NoError(t, err)
	require.Equal(t, int64(1_000), reserves[0].Int64())
	require.Equal(t, int64(2_000), reserves[1].Int64())
	require.Same(t, snapshot.AccruedTotals.AllAssets[borrowXIndex], totals.AllAssets[borrowXIndex])
}

func TestResolveSwapStateAccruesWhenAdapterGetReservesWouldAccrue(t *testing.T) {
	snapshot := trackerSnapshot{
		BlockTimestamp: 1_000 + interestPeriodForSwap + 1,
		Storage: pairStorage{
			ReserveX: big.NewInt(1_000), ReserveY: big.NewInt(2_000),
			LastUpdateTimestamp: 1_000, LastLendingTimestamp: 1_500,
		},
		RawTotals: totalAssetsAndSharesRPC{AllAssets: [6]*big.Int{
			big.NewInt(0), big.NewInt(100), big.NewInt(200),
			big.NewInt(0), big.NewInt(200), big.NewInt(150),
		}},
		AccruedTotals: totalAssetsAndSharesRPC{AllAssets: [6]*big.Int{
			big.NewInt(0), big.NewInt(107), big.NewInt(203),
			big.NewInt(0), big.NewInt(212), big.NewInt(154),
		}},
	}

	reserves, totals, err := resolveSwapState(snapshot)
	require.NoError(t, err)
	// X LP interest = (212 - 200) - (107 - 100) = 5.
	// Y LP interest = (154 - 150) - (203 - 200) = 1.
	require.Equal(t, int64(1_005), reserves[0].Int64())
	require.Equal(t, int64(2_001), reserves[1].Int64())
	require.Same(t, snapshot.AccruedTotals.AllAssets[borrowXIndex], totals.AllAssets[borrowXIndex])
}

func TestDecodePairStorageV110(t *testing.T) {
	pack := func(low, middle *big.Int, high uint32) common.Hash {
		word := new(big.Int).Set(low)
		word.Or(word, new(big.Int).Lsh(new(big.Int).Set(middle), 112))
		word.Or(word, new(big.Int).Lsh(new(big.Int).SetUint64(uint64(high)), 224))
		return common.BigToHash(word)
	}

	storage := decodePairStorageV110(
		pack(big.NewInt(11), big.NewInt(22), 33),
		pack(big.NewInt(44), big.NewInt(55), 66),
	)
	require.Equal(t, int64(11), storage.ReserveX.Int64())
	require.Equal(t, int64(22), storage.ReserveY.Int64())
	require.Equal(t, uint32(33), storage.LastUpdateTimestamp)
	require.Equal(t, int64(44), storage.ReferenceReserveX.Int64())
	require.Equal(t, int64(55), storage.ReferenceReserveY.Int64())
	require.Equal(t, uint32(66), storage.LastLendingTimestamp)
}

func TestUpdatePoolFromSnapshotDoesNotOverwriteNewerBlock(t *testing.T) {
	factory := common.HexToAddress("0x1111111111111111111111111111111111111111")
	beacon := common.HexToAddress("0x2222222222222222222222222222222222222222")
	implementation := common.HexToAddress("0x3333333333333333333333333333333333333333")
	want := entity.Pool{BlockNumber: 11, Timestamp: 99, Reserves: entity.PoolReserves{"7", "8"}, Extra: `{"kept":true}`}
	got, err := updatePoolFromSnapshot(want, &Config{FactoryAddress: factory.Hex()}, StaticExtra{}, trackerSnapshot{
		BlockNumber:    10,
		Beacon:         beacon,
		Implementation: implementation,
	}, 100, want.Timestamp)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestUpdatePoolFromSnapshotScheduledRefreshPreservesActivityTimestamp(t *testing.T) {
	factory := common.HexToAddress("0x1111111111111111111111111111111111111111")
	beacon := common.HexToAddress("0x2222222222222222222222222222222222222222")
	implementation := common.HexToAddress("0x3333333333333333333333333333333333333333")
	snapshot := trackerSnapshot{
		BlockNumber: 10, Beacon: beacon, Implementation: implementation,
		BlockTimestamp: 1_000,
		Storage:        pairStorage{ReserveX: big.NewInt(1_000), ReserveY: big.NewInt(2_000), LastLendingTimestamp: 900},
		References: referenceReservesRPC{
			ReferenceReserveX: big.NewInt(900), ReferenceReserveY: big.NewInt(1_800),
		},
		RawTotals: totalAssetsAndSharesRPC{AllAssets: [6]*big.Int{
			big.NewInt(0), big.NewInt(10), big.NewInt(8),
			big.NewInt(0), big.NewInt(40), big.NewInt(5),
		}},
		AccruedTotals: totalAssetsAndSharesRPC{AllAssets: [6]*big.Int{
			big.NewInt(0), big.NewInt(10), big.NewInt(8),
			big.NewInt(0), big.NewInt(40), big.NewInt(5),
		}},
	}
	p := entity.Pool{BlockNumber: 9, Timestamp: 123, Reserves: entity.PoolReserves{"1", "2"}}
	got, err := updatePoolFromSnapshot(p, &Config{FactoryAddress: factory.Hex()}, StaticExtra{}, snapshot, 1_000, p.Timestamp)
	require.NoError(t, err)
	require.Equal(t, int64(123), got.Timestamp)
	require.Equal(t, int64(1_300), (&PoolTracker{}).NextUpdateAt(got))
}

func TestPoolTrackerActivityTimestampIgnoresScheduledZeroLog(t *testing.T) {
	p := entity.Pool{Timestamp: 1_000}
	headers := map[uint64]entity.BlockHeader{100: {Timestamp: 2_000}}
	require.Equal(t, int64(1_000), activityTimestamp(p, pool.GetNewPoolStateParams{
		Logs: []ethtypes.Log{{}}, BlockHeaders: headers,
	}))
	require.Equal(t, int64(2_000), activityTimestamp(p, pool.GetNewPoolStateParams{
		Logs: []ethtypes.Log{{Address: common.HexToAddress("0x1"), BlockNumber: 100}}, BlockHeaders: headers,
	}))
}

func TestReconcileTrackerMetadataLearnsBlankValuesAndRejectsMismatch(t *testing.T) {
	factory := common.HexToAddress("0x1111111111111111111111111111111111111111")
	beacon := common.HexToAddress("0x2222222222222222222222222222222222222222")
	implementation := common.HexToAddress("0x3333333333333333333333333333333333333333")
	other := common.HexToAddress("0x4444444444444444444444444444444444444444")

	got, err := reconcileTrackerMetadata(&Config{FactoryAddress: factory.Hex()}, StaticExtra{}, beacon, implementation)
	require.NoError(t, err)
	require.Equal(t, factory.Hex(), got.FactoryAddress)
	require.Equal(t, beacon.Hex(), got.PairBeacon)
	require.Equal(t, implementation.Hex(), got.PairImplementation)

	for name, tc := range map[string]struct {
		config *Config
		static StaticExtra
	}{
		"static factory": {
			config: &Config{FactoryAddress: factory.Hex()},
			static: StaticExtra{FactoryAddress: other.Hex()},
		},
		"configured beacon": {
			config: &Config{FactoryAddress: factory.Hex(), PairBeacon: other.Hex()},
		},
		"stored beacon": {
			config: &Config{FactoryAddress: factory.Hex()},
			static: StaticExtra{PairBeacon: other.Hex()},
		},
		"configured implementation": {
			config: &Config{FactoryAddress: factory.Hex(), PairImplementation: other.Hex()},
		},
		"stored implementation upgrade": {
			config: &Config{FactoryAddress: factory.Hex()},
			static: StaticExtra{PairImplementation: other.Hex()},
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := reconcileTrackerMetadata(tc.config, tc.static, beacon, implementation)
			require.Error(t, err)
		})
	}
}

func TestNewPoolTrackerRequiresPinnedImplementationForStorageLayout(t *testing.T) {
	_, err := NewPoolTracker(&Config{}, nil)
	require.Error(t, err)

	tracker, err := NewPoolTracker(&Config{
		PairImplementation: "0x3333333333333333333333333333333333333333",
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, tracker)
}

func TestPoolTrackerNextUpdateAt(t *testing.T) {
	p := entity.Pool{Timestamp: 1_000}
	require.Equal(t, int64(1_090), (&PoolTracker{config: &Config{RefreshInterval: 90 * time.Second}}).NextUpdateAt(p))
	require.Equal(t, int64(1_300), (&PoolTracker{config: &Config{}}).NextUpdateAt(p))
	require.Equal(t, int64(1_300), (&PoolTracker{}).NextUpdateAt(p))
}

func TestTrackerABIMultiReturnDecodeShape(t *testing.T) {
	require.Contains(t, pairABI.Methods, "referenceReserves")
	require.Contains(t, pairABI.Methods, "totalAssetsAndShares")

	referencesData, err := pairABI.Methods["referenceReserves"].Outputs.Pack(big.NewInt(44), big.NewInt(55))
	require.NoError(t, err)
	var references referenceReservesRPC
	require.NoError(t, pairABI.UnpackIntoInterface(&references, "referenceReserves", referencesData))
	require.Equal(t, int64(44), references.ReferenceReserveX.Int64())
	require.Equal(t, int64(55), references.ReferenceReserveY.Int64())

	assets := [6]*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4), big.NewInt(5), big.NewInt(6)}
	shares := [6]*big.Int{big.NewInt(7), big.NewInt(8), big.NewInt(9), big.NewInt(10), big.NewInt(11), big.NewInt(12)}
	totalsData, err := pairABI.Methods["totalAssetsAndShares"].Outputs.Pack(assets, shares)
	require.NoError(t, err)
	var totals totalAssetsAndSharesRPC
	require.NoError(t, pairABI.UnpackIntoInterface(&totals, "totalAssetsAndShares", totalsData))
	require.Equal(t, int64(5), totals.AllAssets[4].Int64())
	require.Equal(t, int64(9), totals.AllShares[2].Int64())
}
