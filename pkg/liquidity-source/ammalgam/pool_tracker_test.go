package ammalgam

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func TestUpdatePoolFromSnapshotDerivesMissingAssets(t *testing.T) {
	factory := common.HexToAddress("0x1111111111111111111111111111111111111111")
	beacon := common.HexToAddress("0x2222222222222222222222222222222222222222")
	implementation := common.HexToAddress("0x3333333333333333333333333333333333333333")
	snapshot := trackerSnapshot{
		BlockNumber:    10,
		Beacon:         beacon,
		Implementation: implementation,
		Reserves: reservesRPC{
			ReserveX:            big.NewInt(1_000),
			ReserveY:            big.NewInt(2_000),
			LastUpdateTimestamp: 77,
		},
		References: referenceReservesRPC{
			ReferenceReserveX: big.NewInt(900),
			ReferenceReserveY: big.NewInt(1_800),
		},
		Totals: totalAssetsAndSharesRPC{
			AllAssets: [6]*big.Int{
				big.NewInt(0), big.NewInt(10), big.NewInt(8),
				big.NewInt(0), big.NewInt(40), big.NewInt(5),
			},
			AllShares: [6]*big.Int{
				big.NewInt(0), big.NewInt(0), big.NewInt(0),
				big.NewInt(0), big.NewInt(0), big.NewInt(0),
			},
		},
	}
	p := entity.Pool{BlockNumber: 9, Timestamp: 1, Reserves: entity.PoolReserves{"1", "2"}}
	got, err := updatePoolFromSnapshot(p, &Config{FactoryAddress: factory.Hex()}, StaticExtra{}, snapshot, 1_700_000_000)
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

	var staticExtra StaticExtra
	require.NoError(t, json.Unmarshal([]byte(got.StaticExtra), &staticExtra))
	require.Equal(t, "0x1111111111111111111111111111111111111111", staticExtra.FactoryAddress)
	require.Equal(t, "0x2222222222222222222222222222222222222222", staticExtra.PairBeacon)
	require.Equal(t, "0x3333333333333333333333333333333333333333", staticExtra.PairImplementation)
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
	}, 100)
	require.NoError(t, err)
	require.Equal(t, want, got)
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

func TestPoolTrackerNextUpdateAt(t *testing.T) {
	p := entity.Pool{Timestamp: 1_000}
	require.Equal(t, int64(1_090), (&PoolTracker{config: &Config{RefreshInterval: 90 * time.Second}}).NextUpdateAt(p))
	require.Equal(t, int64(1_300), (&PoolTracker{config: &Config{}}).NextUpdateAt(p))
	require.Equal(t, int64(1_300), (&PoolTracker{}).NextUpdateAt(p))
}

func TestTrackerABIMultiReturnDecodeShape(t *testing.T) {
	require.Contains(t, pairABI.Methods, "getReserves")
	require.Contains(t, pairABI.Methods, "referenceReserves")
	require.Contains(t, pairABI.Methods, "totalAssetsAndShares")

	reservesData, err := pairABI.Methods["getReserves"].Outputs.Pack(big.NewInt(11), big.NewInt(22), uint32(33))
	require.NoError(t, err)
	var reserves reservesRPC
	require.NoError(t, pairABI.UnpackIntoInterface(&reserves, "getReserves", reservesData))
	require.Equal(t, int64(11), reserves.ReserveX.Int64())
	require.Equal(t, int64(22), reserves.ReserveY.Int64())
	require.Equal(t, uint32(33), reserves.LastUpdateTimestamp)

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
