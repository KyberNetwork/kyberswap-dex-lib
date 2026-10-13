package tickermemev4

import (
	"context"
	"math/big"
	"net/http/httptest"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const snapshotBlock = 112361144

type trackingRPC struct {
	t              *testing.T
	implementation common.Address
	output         []byte
	calls          int
}

func (s *trackingRPC) GetStorageAt(_ context.Context, address common.Address, slot common.Hash, block string) (hexutil.Bytes, error) {
	require.Equal(s.t, HookAddress, address)
	require.Equal(s.t, implementationSlot, slot)
	require.Equal(s.t, hexutil.EncodeUint64(snapshotBlock), block)
	return common.BytesToHash(s.implementation.Bytes()).Bytes(), nil
}

func (s *trackingRPC) GetCode(_ context.Context, _ common.Address, _ string) (hexutil.Bytes, error) {
	s.t.Error("Track must not fetch implementation bytecode")
	return nil, nil
}

func (s *trackingRPC) Call(_ context.Context, call map[string]json.RawMessage, block string, overrides *map[string]json.RawMessage) (hexutil.Bytes, error) {
	require.Equal(s.t, hexutil.EncodeUint64(snapshotBlock), block)
	var to string
	require.NoError(s.t, json.Unmarshal(call["to"], &to))
	require.Equal(s.t, hexutil.Encode(HookAddress[:]), to)
	data := call["input"]
	if len(data) == 0 {
		data = call["data"]
	}
	var input hexutil.Bytes
	require.NoError(s.t, json.Unmarshal(data, &input))
	require.Equal(s.t, crypto.Keccak256([]byte("launches(bytes32)"))[:4], []byte(input[:4]))
	require.Len(s.t, input, 36)
	s.calls++
	return s.output, nil
}

func trackContext(t *testing.T) (*Hook, *uniswapv4.HookParam, *trackingRPC) {
	t.Helper()
	output := make([]byte, 13*32)
	for index, value := range map[int]int64{0: 1, 7: 133, 8: 3000, 9: 5000, 10: 100, 11: 300, 12: 1} {
		copy(output[index*32:(index+1)*32], common.BigToHash(big.NewInt(value)).Bytes())
	}
	service := &trackingRPC{t: t, implementation: ImplementationAddress, output: output}
	server := rpc.NewServer()
	require.NoError(t, server.RegisterName("eth", service))
	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)
	param := &uniswapv4.HookParam{Cfg: &uniswapv4.Config{ChainID: valueobject.ChainIDMonad}, HookAddress: HookAddress, RpcClient: ethrpc.New(httpServer.URL), BlockNumber: big.NewInt(snapshotBlock), Pool: &entity.Pool{Address: "0x1111111111111111111111111111111111111111111111111111111111111111"}}
	return NewHook(param).(*Hook), param, service
}

func TestTrackUsesPonsAndRejectsUpgrade(t *testing.T) {
	h, param, service := trackContext(t)
	data, err := h.Track(context.Background(), param)
	require.NoError(t, err)
	require.JSONEq(t, `{"r":true,"f":233}`, string(data))
	require.EqualValues(t, 233, h.FeeBps)
	_, err = h.Track(context.Background(), param)
	require.NoError(t, err)
	require.Equal(t, 1, service.calls, "reuse Pons's immutable pool fees")
	service.implementation = common.HexToAddress("0xdead")
	_, err = h.Track(context.Background(), param)
	require.ErrorIs(t, err, ErrUnsupportedDeployment)
	require.EqualValues(t, 233, h.FeeBps)

	param.Overrides = map[common.Address]gethclient.OverrideAccount{HookAddress: {StateDiff: map[common.Hash]common.Hash{implementationSlot: common.BytesToHash(ImplementationAddress.Bytes())}}}
	_, err = h.Track(context.Background(), param)
	require.NoError(t, err)
	param.Overrides[ImplementationAddress] = gethclient.OverrideAccount{Code: []byte{0}}
	_, err = h.Track(context.Background(), param)
	require.ErrorIs(t, err, ErrUnsupportedDeployment)
	delete(param.Overrides, ImplementationAddress)
	param.Overrides[HookAddress] = gethclient.OverrideAccount{Code: []byte{0}}
	_, err = h.Track(context.Background(), param)
	require.ErrorIs(t, err, ErrUnsupportedDeployment)
	param.Overrides = map[common.Address]gethclient.OverrideAccount{HookAddress: {State: map[common.Hash]common.Hash{}}}
	_, err = h.Track(context.Background(), param)
	require.ErrorIs(t, err, ErrUnsupportedDeployment)
}
