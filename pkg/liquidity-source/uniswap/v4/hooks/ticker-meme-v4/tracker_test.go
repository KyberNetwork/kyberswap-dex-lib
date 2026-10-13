package tickermemev4

import (
	"context"
	"math/big"
	"net/http/httptest"
	"os"
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
	t               *testing.T
	implementation  common.Address
	code            []byte
	output          []byte
	calls           int
	expectOverrides bool
}

func (s *trackingRPC) GetStorageAt(_ context.Context, address common.Address, slot common.Hash, block string) (hexutil.Bytes, error) {
	require.Equal(s.t, HookAddress, address)
	require.Equal(s.t, implementationSlot, slot)
	require.Equal(s.t, hexutil.EncodeUint64(snapshotBlock), block)
	return common.BytesToHash(s.implementation.Bytes()).Bytes(), nil
}

func (s *trackingRPC) GetCode(_ context.Context, address common.Address, block string) (hexutil.Bytes, error) {
	require.Equal(s.t, ImplementationAddress, address)
	require.Equal(s.t, hexutil.EncodeUint64(snapshotBlock), block)
	return s.code, nil
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
	require.Equal(s.t, hookABI.Methods["launches"].ID, []byte(input[:4]))
	require.Len(s.t, input, 36)
	if s.expectOverrides {
		require.NotNil(s.t, overrides)
	}
	s.calls++
	return s.output, nil
}

func trackContext(t *testing.T) (*Hook, *uniswapv4.HookParam, *trackingRPC) {
	t.Helper()
	// Use a tiny mock runtime for serial RPC tests; the live test checks the real
	// deployment hash without requiring a checked-in bytecode dump.
	code := []byte{0x60, 0x00, 0x60, 0x00, 0xf3}
	originalHash := ImplementationCodeHash
	ImplementationCodeHash = crypto.Keccak256Hash(code)
	t.Cleanup(func() { ImplementationCodeHash = originalHash })
	output, err := hookABI.Methods["launches"].Outputs.Pack(true, false, common.HexToAddress("0xf000000000000000000000000000000000000000"), common.Address{}, common.Address{}, common.Address{}, common.Address{}, uint16(133), uint16(3000), uint16(5000), uint16(100), uint16(300), true)
	require.NoError(t, err)
	implementationVerified.Store(false)
	service := &trackingRPC{t: t, implementation: ImplementationAddress, code: code, output: output}
	server := rpc.NewServer()
	require.NoError(t, server.RegisterName("eth", service))
	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)
	param := &uniswapv4.HookParam{Cfg: &uniswapv4.Config{ChainID: valueobject.ChainIDMonad}, HookAddress: HookAddress, RpcClient: ethrpc.New(httpServer.URL), BlockNumber: big.NewInt(snapshotBlock), Pool: &entity.Pool{Address: "0x1111111111111111111111111111111111111111111111111111111111111111"}}
	return NewHook(param).(*Hook), param, service
}

func TestTrackPinnedStateAndUpgradeRejection(t *testing.T) {
	h, param, service := trackContext(t)
	data, err := h.Track(context.Background(), param)
	require.NoError(t, err)
	var extra Extra
	require.NoError(t, json.Unmarshal(data, &extra))
	require.Equal(t, terms(100, 133), extra)
	require.Equal(t, extra, h.Extra)
	require.EqualValues(t, snapshotBlock, param.Pool.BlockNumber, "published pool block must match validated RPC state")
	_, err = h.Track(context.Background(), param)
	require.NoError(t, err)
	require.Equal(t, 2, service.calls, "registered pools must still refresh after an upgrade")
	service.implementation = common.HexToAddress("0xdead")
	_, err = h.Track(context.Background(), param)
	require.ErrorIs(t, err, ErrUnsupportedDeployment)
	require.Equal(t, extra, h.Extra, "failed tracking changed the last good state")
	require.EqualValues(t, snapshotBlock, param.Pool.BlockNumber, "failed tracking changed the published block")
}

// The pinned implementation address has immutable code: the second refresh must
// not fetch it again, while an overridden code is always hashed.
func TestTrackHashesImplementationCodeOnce(t *testing.T) {
	h, param, service := trackContext(t)
	_, err := h.Track(context.Background(), param)
	require.NoError(t, err)
	service.code = []byte{0}
	_, err = h.Track(context.Background(), param)
	require.NoError(t, err, "verified implementation code must not be fetched again")
	param.Overrides = map[common.Address]gethclient.OverrideAccount{ImplementationAddress: {Code: []byte{0}}}
	_, err = h.Track(context.Background(), param)
	require.ErrorIs(t, err, ErrUnsupportedDeployment, "overridden code must always be hashed")
}

func TestTrackOverridesAndInvalidSnapshots(t *testing.T) {
	h, param, service := trackContext(t)
	param.Overrides = map[common.Address]gethclient.OverrideAccount{
		HookAddress:           {StateDiff: map[common.Hash]common.Hash{implementationSlot: common.BytesToHash(ImplementationAddress.Bytes())}},
		ImplementationAddress: {Code: service.code},
	}
	service.expectOverrides = true
	_, err := h.Track(context.Background(), param)
	require.NoError(t, err)
	param.Overrides[ImplementationAddress] = gethclient.OverrideAccount{Code: []byte{0}}
	_, err = h.Track(context.Background(), param)
	require.ErrorIs(t, err, ErrUnsupportedDeployment)
	param.Overrides = map[common.Address]gethclient.OverrideAccount{HookAddress: {State: map[common.Hash]common.Hash{}}}
	_, err = h.Track(context.Background(), param)
	require.ErrorIs(t, err, ErrUnsupportedDeployment, "full state override clears absent slots")
	param.Overrides = map[common.Address]gethclient.OverrideAccount{HookAddress: {Code: []byte{0}}}
	_, err = h.Track(context.Background(), param)
	require.ErrorIs(t, err, ErrUnsupportedDeployment)
	param.Overrides = nil
	service.code = []byte{0}
	_, err = h.Track(context.Background(), param)
	require.ErrorIs(t, err, ErrUnsupportedDeployment)
	param.BlockNumber = nil
	_, err = h.Track(context.Background(), param)
	require.Error(t, err)
}

func TestTrackLiveDeployment(t *testing.T) {
	url := os.Getenv("TICKER_MEME_RPC")
	if url == "" {
		t.Skip("set TICKER_MEME_RPC to verify the pinned Monad deployment")
	}
	param := &uniswapv4.HookParam{Cfg: &uniswapv4.Config{ChainID: valueobject.ChainIDMonad}, HookAddress: HookAddress, RpcClient: ethrpc.New(url), BlockNumber: big.NewInt(snapshotBlock), Pool: &entity.Pool{Address: common.Hash{}.Hex()}}
	h := NewHook(param).(*Hook)
	_, err := h.Track(context.Background(), param)
	require.NoError(t, err)
	require.False(t, h.Registered)
	require.Equal(t, ImplementationCodeHash, h.ImplementationHash)
}
