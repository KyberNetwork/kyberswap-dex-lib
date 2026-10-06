package flywheelfun

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var aggregateABI = mustABI(`[{"type":"function","name":"aggregate","inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"returnData","type":"bytes[]"}]}]`)

type mockChain struct {
	t         *testing.T
	header    *types.Header
	quote     common.Address
	graduated bool
	fail      bool
	chain     uint64
	calls     int
}

func (m *mockChain) ChainId() hexutil.Uint64 { return hexutil.Uint64(m.chain) }
func (m *mockChain) GetBlockByNumber(context.Context, string, bool) (*types.Header, error) {
	return m.header, nil
}
func (m *mockChain) Call(_ context.Context, arg map[string]json.RawMessage, block string) (hexutil.Bytes, error) {
	m.calls++
	require.Equal(m.t, hexutil.EncodeBig(m.header.Number), block, "all dependent reads must use the snapshot block")
	if m.fail {
		return nil, errors.New("fixture RPC failure")
	}
	var data hexutil.Bytes
	payload := arg["input"]
	if len(payload) == 0 {
		payload = arg["data"]
	}
	require.NoError(m.t, json.Unmarshal(payload, &data))
	return m.execute(data)
}
func (m *mockChain) execute(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return nil, errors.New("short call")
	}
	if method, err := aggregateABI.MethodById(data[:4]); err == nil {
		v, err := method.Inputs.Unpack(data[4:])
		if err != nil {
			return nil, err
		}
		calls := *abi.ConvertType(v[0], new([]struct {
			Target   common.Address
			CallData []byte
		})).(*[]struct {
			Target   common.Address
			CallData []byte
		})
		results := make([][]byte, len(calls))
		for i, c := range calls {
			results[i], err = m.execute(c.CallData)
			if err != nil {
				return nil, err
			}
		}
		return method.Outputs.Pack(m.header.Number, results)
	}
	method, err := factoryABI.MethodById(data[:4])
	if err != nil {
		method, err = tokenABI.MethodById(data[:4])
	}
	if err != nil {
		method, err = stateABI.MethodById(data[:4])
	}
	if err != nil {
		return nil, err
	}
	switch method.Name {
	case "allLaunchesCount":
		return method.Outputs.Pack(big.NewInt(1))
	case "allLaunches":
		return method.Outputs.Pack(common.HexToAddress(testEntity().Address))
	case "quoteAsset":
		return method.Outputs.Pack(m.quote)
	case "factory":
		return method.Outputs.Pack(common.HexToAddress(Factory))
	case "nativeMarketAdapter":
		return method.Outputs.Pack(common.HexToAddress(MarketAdapter))
	case "balanceOf":
		return method.Outputs.Pack(big.NewInt(100000000))
	case "extsload": // V4 slot0 with sqrtPriceX96 = 2^97, i.e. 4 quote per WETH
		return method.Outputs.Pack([][32]byte{common.BigToHash(new(big.Int).Lsh(big.NewInt(1), 97))})
	case "curves":
		zero := common.Address{}
		return method.Outputs.Pack(zero, m.quote, big.NewInt(100000), big.NewInt(100000000), big.NewInt(50000), big.NewInt(1000000), big.NewInt(1), big.NewInt(0), uint16(0), uint16(295), uint16(95), uint16(0), m.graduated, zero, zero, zero, big.NewInt(0))
	}
	return nil, errors.New("unexpected method " + method.Name)
}
func fixtureRPC(t *testing.T) (*ethrpc.Client, *mockChain) {
	t.Helper()
	m := &mockChain{t: t, chain: 4663, quote: common.HexToAddress(WETH), header: &types.Header{Number: big.NewInt(100), Difficulty: big.NewInt(0), Time: 1000, GasLimit: 30000000}}
	server := rpc.NewServer()
	require.NoError(t, server.RegisterName("eth", m))
	client := rpc.DialInProc(server)
	t.Cleanup(client.Close)
	t.Cleanup(server.Stop)
	return ethrpc.NewWithClient(ethclient.NewClient(client)).SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11")), m
}
func TestDiscoveryTrackingAndUnverifiedGraduationRejected(t *testing.T) {
	client, m := fixtureRPC(t)
	ctx := context.Background()
	list := NewPoolsListUpdater(&Config{}, client)
	pools, cursor, err := list.GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.Equal(t, []string{"0", "0"}, []string(pools[0].Reserves))
	require.Empty(t, pools[0].Tokens[0].Symbol)
	empty, _, err := list.GetNewPools(ctx, cursor)
	require.NoError(t, err)
	require.Empty(t, empty)
	tracker, err := NewPoolTracker(&Config{}, client)
	require.NoError(t, err)
	p, err := tracker.GetNewPoolState(ctx, pools[0], pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.Equal(t, uint64(100), p.BlockNumber)
	s, err := NewPoolSimulator(p)
	require.NoError(t, err)
	require.Equal(t, testCurve(), s.Curve)
	m.graduated = true
	_, err = tracker.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
	require.ErrorIs(t, err, ErrState)
}
func TestDiscoveryReorgReplaysAndFailureKeepsCursor(t *testing.T) {
	client, m := fixtureRPC(t)
	ctx := context.Background()
	list := NewPoolsListUpdater(&Config{NewPoolLimit: 1}, client)
	_, cursor, err := list.GetNewPools(ctx, nil)
	require.NoError(t, err)
	m.header.Extra = []byte("replacement block")
	p, _, err := list.GetNewPools(ctx, cursor)
	require.NoError(t, err)
	require.Len(t, p, 1)
	m.fail = true
	_, after, err := list.GetNewPools(ctx, cursor)
	require.Error(t, err)
	require.Equal(t, cursor, after)
}
func TestWrongChainAndMetadataRejected(t *testing.T) {
	client, m := fixtureRPC(t)
	list := NewPoolsListUpdater(&Config{}, client)
	_, _, err := list.GetNewPools(context.Background(), []byte("not json"))
	require.Error(t, err)
	m.chain = 1
	_, _, err = list.GetNewPools(context.Background(), nil)
	require.ErrorIs(t, err, ErrState)
	require.Equal(t, 0, m.calls)
}
func TestCustomPairDiscoveredButNotMisquoted(t *testing.T) {
	client, m := fixtureRPC(t)
	m.quote = common.HexToAddress("0x73c2de14c7fa0a57cc2d9722b959ea70b881ffe4")
	list := NewPoolsListUpdater(&Config{}, client)
	p, _, err := list.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, p, 1)
	require.Contains(t, p[0].StaticExtra, strings.ToLower(m.quote.Hex()))
	tracker, _ := NewPoolTracker(&Config{}, client)
	tracked, err := tracker.GetNewPoolState(context.Background(), p[0], pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	_, err = NewPoolSimulator(tracked)
	require.ErrorIs(t, err, ErrUnsupported)
}

// Listing copies the configured base pool into StaticExtra (router-service reads basePools
// from it), and the tracker values the quote reserve in WETH at the hop's spot price.
func TestRoutedPairListedWithBasePools(t *testing.T) {
	client, m := fixtureRPC(t)
	boomer := common.HexToAddress("0x73c2de14c7fa0a57cc2d9722b959ea70b881ffe4")
	m.quote = boomer
	id := "0x8742f10cc122395bfb79eb2dc51fb15b97b297759a5f79b5f2463d38af59083d"
	list := NewPoolsListUpdater(&Config{QuoteBasePools: map[string]string{boomer.Hex(): id}}, client)
	p, _, err := list.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	var st StaticExtra
	require.NoError(t, json.Unmarshal([]byte(p[0].StaticExtra), &st))
	require.Equal(t, []string{id}, st.BasePools)
	tracker, _ := NewPoolTracker(nil, client)
	tracked, err := tracker.GetNewPoolState(context.Background(), p[0], pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.Equal(t, []string{"12500", "100000000"}, []string(tracked.Reserves), "50000 BOOMER of real quote = 12500 WETH")
	_, err = NewPoolSimulator(tracked)
	require.ErrorIs(t, err, ErrBasePool)
}
