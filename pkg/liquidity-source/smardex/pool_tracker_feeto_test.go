package smardex

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

const (
	feeToTestPair    = "0x00000000000000000000000000000000000000aa"
	feeToTestFactory = "0x00000000000000000000000000000000000000fa"
)

// feeToStub answers the pair's views with fixed state, factory() with the factory, and the factory's feeTo() with
// feeTo; a nil factoryOK or feeToOK makes that call fail.
func feeToStub(t *testing.T, feeTo common.Address, factoryOK, feeToOK bool) *testutil.MulticallStub {
	t.Helper()
	pack := func(a abi.ABI, method string, out ...any) []byte {
		b, err := a.Methods[method].Outputs.Pack(out...)
		require.NoError(t, err)
		return b
	}
	one := big.NewInt(1e18)
	return testutil.NewMulticallStub(t, func(target common.Address, data []byte) (bool, []byte) {
		if target == common.HexToAddress(feeToTestFactory) {
			m, err := factoryABI.MethodById(data)
			require.NoError(t, err)
			require.Equal(t, factoryFeeToMethod, m.Name)
			return feeToOK, pack(factoryABI, factoryFeeToMethod, feeTo)
		}
		require.Equal(t, common.HexToAddress(feeToTestPair), target)
		m, err := pairABI.MethodById(data)
		require.NoError(t, err)
		switch m.Name {
		case pairFactoryMethod:
			return factoryOK, pack(pairABI, m.Name, common.HexToAddress(feeToTestFactory))
		case pairGetPriceAverageMethod:
			return true, pack(pairABI, m.Name, one, one, big.NewInt(1))
		case pairGetPairFeesMethod:
			return true, pack(pairABI, m.Name, big.NewInt(700), big.NewInt(200))
		default:
			return true, pack(pairABI, m.Name, one, one)
		}
	})
}

func feeToTrack(t *testing.T, stub *testutil.MulticallStub, p entity.Pool) (entity.Pool, SmardexPair) {
	t.Helper()
	tracker, err := NewPoolTracker(&Config{DexID: DexTypeSmardex, ChainID: 42161}, stub.Client())
	require.NoError(t, err)
	got, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	var extra SmardexPair
	require.NoError(t, json.Unmarshal([]byte(got.Extra), &extra))
	return got, extra
}

// The first refresh reads the pair's factory in its batch and stores it, the fee kept on; later refreshes read the
// factory's feeTo in the same batch, so FeeOff follows it. Every call is in the one multicall.
func TestPoolTracker_FeeToFollowsTheFactory(t *testing.T) {
	p := entity.Pool{Address: feeToTestPair, Tokens: []*entity.PoolToken{{Address: "0x01"}, {Address: "0x02"}}}

	stub := feeToStub(t, common.Address{}, true, true)
	first, extra := feeToTrack(t, stub, p)
	require.Equal(t, common.HexToAddress(feeToTestFactory), extra.Factory)
	require.False(t, extra.FeeOff, "feeTo not read yet: the fee stays on")
	require.Len(t, stub.Targets(), 1, "one eth_call")

	for name, tc := range map[string]struct {
		feeTo  common.Address
		ok     bool
		feeOff bool
	}{
		"feeTo set":        {common.HexToAddress("0x01"), true, false},
		"no feeTo":         {common.Address{}, true, true},
		"feeTo read fails": {common.Address{}, false, false},
	} {
		t.Run(name, func(t *testing.T) {
			stub := feeToStub(t, tc.feeTo, true, tc.ok)
			_, extra := feeToTrack(t, stub, first)
			require.Equal(t, tc.feeOff, extra.FeeOff)
			require.Equal(t, common.HexToAddress(feeToTestFactory), extra.Factory, "the factory stays stored")
			targets := stub.Targets()
			require.Len(t, targets, 1, "one eth_call")
			require.Contains(t, targets[0], common.HexToAddress(feeToTestFactory))
		})
	}

	// A failed factory() read keeps the fee on and the refresh going.
	_, extra = feeToTrack(t, feeToStub(t, common.Address{}, false, true), p)
	require.Equal(t, common.Address{}, extra.Factory)
	require.False(t, extra.FeeOff)
}
