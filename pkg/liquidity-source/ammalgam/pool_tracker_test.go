package ammalgam

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var trackerMulticallABI = func() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(`[
		{"type":"function","name":"tryAggregate","stateMutability":"payable","inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]},
		{"type":"function","name":"tryBlockAndAggregate","stateMutability":"payable","inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}
	]`))
	if err != nil {
		panic(err)
	}
	return parsed
}()

type trackerMulticall struct {
	Target   common.Address
	CallData []byte
}

type trackerMulticallResult struct {
	Success    bool
	ReturnData []byte
}

type trackerRPCFixture struct {
	t               *testing.T
	header          *ethtypes.Header
	beacon          common.Address
	callBlocks      []string
	headerBlock     string
	rawAssets       [6]*big.Int
	accruedAssets   [6]*big.Int
	assetsAndShares [6]*big.Int
}

func (m *trackerRPCFixture) GetBlockByNumber(_ context.Context, block string, _ bool) (*ethtypes.Header, error) {
	m.headerBlock = block
	return m.header, nil
}

func (m *trackerRPCFixture) Call(_ context.Context, arg map[string]json.RawMessage, block string) (hexutil.Bytes, error) {
	m.callBlocks = append(m.callBlocks, block)
	var data hexutil.Bytes
	payload := arg["input"]
	if len(payload) == 0 {
		payload = arg["data"]
	}
	require.NoError(m.t, json.Unmarshal(payload, &data))
	return m.execute(data)
}

func (m *trackerRPCFixture) execute(data []byte) ([]byte, error) {
	if method, err := trackerMulticallABI.MethodById(data[:4]); err == nil {
		values, err := method.Inputs.Unpack(data[4:])
		if err != nil {
			return nil, err
		}
		calls := *abi.ConvertType(values[1], new([]trackerMulticall)).(*[]trackerMulticall)
		results := make([]trackerMulticallResult, len(calls))
		for i := range calls {
			results[i].Success = true
			results[i].ReturnData, err = m.execute(calls[i].CallData)
			if err != nil {
				return nil, err
			}
		}
		if method.Name == "tryBlockAndAggregate" {
			return method.Outputs.Pack(m.header.Number, common.HexToHash("0x42"), results)
		}
		return method.Outputs.Pack(results)
	}
	if method, err := factoryABI.MethodById(data[:4]); err == nil && method.Name == "pairBeacon" {
		return method.Outputs.Pack(m.beacon)
	}
	method, err := pairABI.MethodById(data[:4])
	if err != nil {
		return nil, err
	}
	switch method.Name {
	case "getReserves":
		return method.Outputs.Pack(big.NewInt(1_000), big.NewInt(2_000), uint32(1_000))
	case "referenceReserves":
		return method.Outputs.Pack(big.NewInt(900), big.NewInt(1_800))
	case "totalAssetsAndShares":
		values, err := method.Inputs.Unpack(data[4:])
		if err != nil {
			return nil, err
		}
		assets := m.rawAssets
		if values[0].(bool) {
			assets = m.accruedAssets
		}
		return method.Outputs.Pack(assets, m.assetsAndShares)
	default:
		return nil, errors.New("unexpected Ammalgam pair method")
	}
}

func TestPoolTrackerPinsGetterSnapshotAndHeaderToOneBlock(t *testing.T) {
	zeroes := [6]*big.Int{}
	for i := range zeroes {
		zeroes[i] = big.NewInt(0)
	}
	raw := zeroes
	raw[depositXIndex], raw[depositYIndex] = big.NewInt(10), big.NewInt(8)
	raw[borrowXIndex], raw[borrowYIndex] = big.NewInt(40), big.NewInt(5)
	accrued := zeroes
	accrued[depositXIndex], accrued[depositYIndex] = big.NewInt(15), big.NewInt(8)
	accrued[borrowXIndex], accrued[borrowYIndex] = big.NewInt(55), big.NewInt(5)
	fixture := &trackerRPCFixture{
		t: t, header: &ethtypes.Header{
			Number: big.NewInt(42), Difficulty: big.NewInt(0), Time: 1_000 + 86_399, GasLimit: 30_000_000,
		},
		beacon:          common.HexToAddress("0x2222222222222222222222222222222222222222"),
		rawAssets:       raw,
		accruedAssets:   accrued,
		assetsAndShares: zeroes,
	}
	server := rpc.NewServer()
	require.NoError(t, server.RegisterName("eth", fixture))
	client := rpc.DialInProc(server)
	t.Cleanup(client.Close)
	t.Cleanup(server.Stop)
	rpcClient := ethrpc.NewWithClient(ethclient.NewClient(client)).
		SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))
	tracker, err := NewPoolTracker(&Config{
		FactoryAddress: "0x1111111111111111111111111111111111111111",
	}, rpcClient)
	require.NoError(t, err)

	got, err := tracker.GetNewPoolState(t.Context(), entity.Pool{
		Address: testPair, Reserves: entity.PoolReserves{"1", "2"}, StaticExtra: `{}`,
	}, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.Equal(t, uint64(42), got.BlockNumber)
	require.Equal(t, entity.PoolReserves{"1000", "2000"}, got.Reserves)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(got.Extra), &extra))
	require.Equal(t, "40", extra.MissingAssets[0].String())
	require.Equal(t, []string{"latest", "0x2a"}, fixture.callBlocks)
	require.Equal(t, "0x2a", fixture.headerBlock)
}

func TestUpdatePoolFromSnapshotDerivesMissingAssets(t *testing.T) {
	factory := common.HexToAddress("0x1111111111111111111111111111111111111111")
	beacon := common.HexToAddress("0x2222222222222222222222222222222222222222")
	snapshot := trackerSnapshot{
		BlockNumber:    10,
		BlockTimestamp: 100,
		Beacon:         beacon,
		Reserves: getReservesRPC{
			ReserveX:             big.NewInt(1_000),
			ReserveY:             big.NewInt(2_000),
			LastLendingTimestamp: 77,
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
	require.Equal(t, uint32(77), extra.LastLendingTimestamp)
	require.Equal(t, int64(1_700_000_000), extra.LastRefreshTimestamp)

	var staticExtra StaticExtra
	require.NoError(t, json.Unmarshal([]byte(got.StaticExtra), &staticExtra))
	require.Equal(t, "0x1111111111111111111111111111111111111111", staticExtra.FactoryAddress)
	require.Equal(t, "0x2222222222222222222222222222222222222222", staticExtra.PairBeacon)
}

func TestUpdatePoolFromSnapshotSelectsTotalsAtSwapThresholdAndTimestampWrap(t *testing.T) {
	factory := common.HexToAddress("0x1111111111111111111111111111111111111111")
	beacon := common.HexToAddress("0x2222222222222222222222222222222222222222")
	const maxUint32 = uint64(1<<32 - 1)
	for _, tc := range []struct {
		name        string
		last, now   uint64
		wantMissing string
	}{
		{"ordinary/86398", 1_000, 1_000 + 86_398, "30"},
		{"ordinary/86399", 1_000, 1_000 + 86_399, "40"},
		{"ordinary/86400", 1_000, 1_000 + 86_400, "40"},
		{"wrap/86398", maxUint32 - 100, maxUint32 - 100 + 86_398, "30"},
		{"wrap/86399", maxUint32 - 100, maxUint32 - 100 + 86_399, "40"},
		{"wrap/86400", maxUint32 - 100, maxUint32 - 100 + 86_400, "40"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := trackerSnapshot{
				BlockNumber:    10,
				BlockTimestamp: tc.now,
				Beacon:         beacon,
				Reserves: getReservesRPC{
					ReserveX: big.NewInt(1_000), ReserveY: big.NewInt(2_000), LastLendingTimestamp: uint32(tc.last),
				},
				References: referenceReservesRPC{ReferenceReserveX: big.NewInt(900), ReferenceReserveY: big.NewInt(1_800)},
				RawTotals: totalAssetsAndSharesRPC{AllAssets: [6]*big.Int{
					big.NewInt(0), big.NewInt(10), big.NewInt(8), big.NewInt(0), big.NewInt(40), big.NewInt(5),
				}},
				AccruedTotals: totalAssetsAndSharesRPC{AllAssets: [6]*big.Int{
					big.NewInt(0), big.NewInt(15), big.NewInt(8), big.NewInt(0), big.NewInt(55), big.NewInt(5),
				}},
			}
			got, err := updatePoolFromSnapshot(entity.Pool{}, &Config{FactoryAddress: factory.Hex()}, StaticExtra{},
				snapshot, int64(tc.now), 0)
			require.NoError(t, err)
			require.Equal(t, entity.PoolReserves{"1000", "2000"}, got.Reserves,
				"getReserves preview is authoritative; do not reconstruct LP interest")
			var extra Extra
			require.NoError(t, json.Unmarshal([]byte(got.Extra), &extra))
			require.Equal(t, tc.wantMissing, extra.MissingAssets[0].String())
		})
	}
}

func TestUpdatePoolFromSnapshotDoesNotOverwriteNewerBlock(t *testing.T) {
	factory := common.HexToAddress("0x1111111111111111111111111111111111111111")
	beacon := common.HexToAddress("0x2222222222222222222222222222222222222222")
	want := entity.Pool{BlockNumber: 11, Timestamp: 99, Reserves: entity.PoolReserves{"7", "8"}, Extra: `{"kept":true}`}
	got, err := updatePoolFromSnapshot(want, &Config{FactoryAddress: factory.Hex()}, StaticExtra{}, trackerSnapshot{
		BlockNumber: 10,
		Beacon:      beacon,
	}, 100, want.Timestamp)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestUpdatePoolFromSnapshotScheduledRefreshPreservesActivityTimestamp(t *testing.T) {
	factory := common.HexToAddress("0x1111111111111111111111111111111111111111")
	beacon := common.HexToAddress("0x2222222222222222222222222222222222222222")
	snapshot := trackerSnapshot{
		BlockNumber: 10, BlockTimestamp: 1_000, Beacon: beacon,
		Reserves: getReservesRPC{ReserveX: big.NewInt(1_000), ReserveY: big.NewInt(2_000)},
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
	other := common.HexToAddress("0x4444444444444444444444444444444444444444")

	got, err := reconcileTrackerMetadata(&Config{FactoryAddress: factory.Hex()}, StaticExtra{}, beacon)
	require.NoError(t, err)
	require.Equal(t, factory.Hex(), got.FactoryAddress)
	require.Equal(t, beacon.Hex(), got.PairBeacon)

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
	} {
		t.Run(name, func(t *testing.T) {
			_, err := reconcileTrackerMetadata(tc.config, tc.static, beacon)
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
	var reserves getReservesRPC
	require.NoError(t, pairABI.UnpackIntoInterface(&reserves, "getReserves", reservesData))
	require.Equal(t, int64(11), reserves.ReserveX.Int64())
	require.Equal(t, int64(22), reserves.ReserveY.Int64())
	require.Equal(t, uint32(33), reserves.LastLendingTimestamp)

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

// A beacon implementation upgrade must not stop updates, and pools stored with
// the retired pairImplementation field must still decode and update.
func TestOldPairImplementationFieldIgnored(t *testing.T) {
	factory := common.HexToAddress("0x1111111111111111111111111111111111111111")
	beacon := common.HexToAddress("0x2222222222222222222222222222222222222222")
	var staticExtra StaticExtra
	require.NoError(t, json.Unmarshal([]byte(
		`{"factoryAddress":"`+factory.Hex()+`","pairBeacon":"`+beacon.Hex()+
			`","pairImplementation":"0x4c48819118171a2f907e8218a9aee7df5971fbca","g":1}`), &staticExtra))
	_, err := reconcileTrackerMetadata(&Config{FactoryAddress: factory.Hex()}, staticExtra, beacon)
	require.NoError(t, err)
}
