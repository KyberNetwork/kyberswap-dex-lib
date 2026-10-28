package ammalgam

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestPairCreatedDecoder(t *testing.T) {
	factory := common.HexToAddress("0x1111111111111111111111111111111111111111")
	x := common.HexToAddress("0x2222222222222222222222222222222222222222")
	y := common.HexToAddress("0x3333333333333333333333333333333333333333")
	pair := common.HexToAddress("0x4444444444444444444444444444444444444444")
	f := NewPoolFactory(&Config{DexID: "ammalgam-dlex", FactoryAddress: factory.Hex(), Gas: 200000})
	wantTopic := crypto.Keccak256Hash([]byte("PairCreated(address,address,address,uint256)"))
	require.Equal(t, wantTopic, factoryABI.Events["PairCreated"].ID)
	require.Equal(t, []common.Hash{wantTopic}, f.SupportedEventTopics())
	require.True(t, f.IsEventSupported(wantTopic))

	data := append(common.LeftPadBytes(pair.Bytes(), 32), common.LeftPadBytes(big.NewInt(1).Bytes(), 32)...)
	log := ethtypes.Log{Address: factory, Topics: []common.Hash{wantTopic, common.BytesToHash(x.Bytes()), common.BytesToHash(y.Bytes())}, Data: data, BlockNumber: 123}
	p, err := f.DecodePoolCreated(log)
	require.NoError(t, err)
	require.Equal(t, "0x4444444444444444444444444444444444444444", p.Address)
	require.Equal(t, "ammalgam-dlex", p.Exchange)
	require.Equal(t, DexType, p.Type)
	require.Equal(t, uint64(123), p.BlockNumber)
	require.Equal(t, []string{"0", "0"}, []string(p.Reserves))
	require.Equal(t, "0x2222222222222222222222222222222222222222", p.Tokens[0].Address)
	require.Equal(t, "0x3333333333333333333333333333333333333333", p.Tokens[1].Address)
	require.True(t, p.Tokens[0].Swappable && p.Tokens[1].Swappable)
	var meta StaticExtra
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &meta))
	require.Equal(t, "0x1111111111111111111111111111111111111111", meta.FactoryAddress)
	require.Equal(t, int64(200000), meta.Gas)
	var wire map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &wire))
	require.Contains(t, wire, "g")
	require.NotContains(t, wire, "gas")
	addrs, err := f.DecodePoolAddressesFromFactoryLog(context.Background(), log)
	require.NoError(t, err)
	require.Equal(t, []string{p.Address}, addrs)

	for _, bad := range []ethtypes.Log{
		{},
		{Address: y, Topics: log.Topics, Data: data},
		{Address: factory, Topics: []common.Hash{common.Hash{}}, Data: data},
		{Address: factory, Topics: log.Topics[:2], Data: data},
		{Address: factory, Topics: log.Topics, Data: data[:31]},
	} {
		_, err := f.DecodePoolCreated(bad)
		require.Error(t, err)
	}
}

func TestDiscoveryPaginationAndConfig(t *testing.T) {
	require.Equal(t, 3, discoveryBatchSize(10, 7, 3))
	require.Equal(t, 3, discoveryBatchSize(10, 7, 0))
	require.Equal(t, 0, discoveryBatchSize(10, 10, 3))
	require.Equal(t, 2, discoveryBatchSize(10, 7, 2))

	beacon := common.HexToAddress("0x1111111111111111111111111111111111111111")
	implementation := common.HexToAddress("0x2222222222222222222222222222222222222222")
	require.NoError(t, validateFactoryMetadata(&Config{}, beacon, implementation))
	require.NoError(t, validateFactoryMetadata(&Config{PairBeacon: beacon.Hex(), PairImplementation: implementation.Hex()}, beacon, implementation))
	require.Error(t, validateFactoryMetadata(&Config{PairBeacon: implementation.Hex()}, beacon, implementation))
	require.Error(t, validateFactoryMetadata(&Config{PairImplementation: beacon.Hex()}, beacon, implementation))
	require.Error(t, requireAllCallsSucceeded(&ethrpc.Response{Result: []bool{true, false}}, 2))
	require.Error(t, requireAllCallsSucceeded(&ethrpc.Response{Result: []bool{true}}, 2))
}

func TestUnderlyingTokensABI(t *testing.T) {
	x := common.HexToAddress("0x2222222222222222222222222222222222222222")
	y := common.HexToAddress("0x3333333333333333333333333333333333333333")
	data, err := pairABI.Methods["underlyingTokens"].Outputs.Pack(x, y)
	require.NoError(t, err)
	var tokens struct {
		TokenX common.Address
		TokenY common.Address
	}
	require.NoError(t, pairABI.UnpackIntoInterface(&tokens, "underlyingTokens", data))
	require.Equal(t, x, tokens.TokenX)
	require.Equal(t, y, tokens.TokenY)
}
