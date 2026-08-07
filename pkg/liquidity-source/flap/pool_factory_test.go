package flap

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func TestIsEventSupported(t *testing.T) {
	f := NewPoolFactory(&Config{}, nil)

	assert.True(t, f.IsEventSupported(tokenCreatedEventTopic))
	assert.False(t, f.IsEventSupported(flapTokenCirculatingSupplyChangedEventTopic),
		"IsEventSupported must only gate the new-pool-discovery path (GetNewPoolsFromLogs); the other "+
			"two topics are handled by DecodePoolAddressesFromFactoryLog independently")
	assert.False(t, f.IsEventSupported(launchedToDEXEventTopic))
	assert.False(t, f.IsEventSupported(common.HexToHash("0xdeadbeef")))
}

func mustPackEvent(t *testing.T, name string, args ...any) []byte {
	t.Helper()
	data, err := portalABI.Events[name].Inputs.Pack(args...)
	require.NoError(t, err)
	return data
}

func TestDecodePoolAddressesFromFactoryLog(t *testing.T) {
	f := NewPoolFactory(&Config{ChainID: 56}, nil)
	token := common.HexToAddress("0x279c3ef82ed0a886f8fcaa53d7e4215eb9297777")

	t.Run("TokenCreated", func(t *testing.T) {
		data := mustPackEvent(t, "TokenCreated",
			big.NewInt(1786013638),
			common.HexToAddress("0x0e13e6a48bf5d49f678459bf2f93ce688c67afa7"),
			big.NewInt(1),
			token,
			"name", "symbol", "meta",
		)
		addrs, err := f.DecodePoolAddressesFromFactoryLog(context.Background(), ethtypes.Log{
			Topics: []common.Hash{tokenCreatedEventTopic},
			Data:   data,
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"0x279c3ef82ed0a886f8fcaa53d7e4215eb9297777"}, addrs)
	})

	t.Run("FlapTokenCirculatingSupplyChanged", func(t *testing.T) {
		data := mustPackEvent(t, "FlapTokenCirculatingSupplyChanged", token, big.NewInt(123456))
		addrs, err := f.DecodePoolAddressesFromFactoryLog(context.Background(), ethtypes.Log{
			Topics: []common.Hash{flapTokenCirculatingSupplyChangedEventTopic},
			Data:   data,
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"0x279c3ef82ed0a886f8fcaa53d7e4215eb9297777"}, addrs)
	})

	t.Run("LaunchedToDEX", func(t *testing.T) {
		data := mustPackEvent(t, "LaunchedToDEX", token,
			common.HexToAddress("0x1111111111111111111111111111111111aaaa"),
			big.NewInt(1), big.NewInt(2))
		addrs, err := f.DecodePoolAddressesFromFactoryLog(context.Background(), ethtypes.Log{
			Topics: []common.Hash{launchedToDEXEventTopic},
			Data:   data,
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"0x279c3ef82ed0a886f8fcaa53d7e4215eb9297777"}, addrs)
	})

	t.Run("UnsupportedTopic", func(t *testing.T) {
		_, err := f.DecodePoolAddressesFromFactoryLog(context.Background(), ethtypes.Log{
			Topics: []common.Hash{common.HexToHash("0xdeadbeef")},
			Data:   []byte{},
		})
		assert.ErrorIs(t, err, ErrInvalidEvent)
	})

	t.Run("NoTopics", func(t *testing.T) {
		_, err := f.DecodePoolAddressesFromFactoryLog(context.Background(), ethtypes.Log{})
		assert.ErrorIs(t, err, ErrInvalidEvent)
	})

	t.Run("MalformedTokenCreatedData", func(t *testing.T) {
		_, err := f.DecodePoolAddressesFromFactoryLog(context.Background(), ethtypes.Log{
			Topics: []common.Hash{tokenCreatedEventTopic},
			Data:   []byte{0x01, 0x02},
		})
		assert.Error(t, err)
	})
}

func TestDecodePoolCreated_MalformedData(t *testing.T) {
	f := NewPoolFactory(&Config{}, nil)
	_, err := f.DecodePoolCreated(ethtypes.Log{Data: []byte{0x01}})
	assert.Error(t, err)
}

// rpcResult builds the raw eth_call response bytes for a TokenStateV8-shaped tuple: a Tradable, non-tax
// token with the given quote token address, mirroring live values verified earlier against BSC mainnet.
func encodeTokenStateV8(quoteToken common.Address) string {
	words := make([]byte, 0, 18*32)
	push := func(n *big.Int) {
		b := make([]byte, 32)
		n.FillBytes(b)
		words = append(words, b...)
	}
	pushAddr := func(a common.Address) {
		b := make([]byte, 32)
		copy(b[12:], a.Bytes())
		words = append(words, b...)
	}

	push(big.NewInt(1))                                       // status = Tradable
	push(big.NewInt(1_000000000000000000))                    // reserve
	push(new(big.Int).Mul(big.NewInt(3e8), big.NewInt(1e18))) // circulatingSupply
	push(big.NewInt(1))                                       // price
	push(big.NewInt(5))                                       // tokenVersion
	push(big.NewInt(5000000000000000000))                     // r
	push(new(big.Int).Mul(big.NewInt(1e8), big.NewInt(1e18))) // h
	push(new(big.Int).Mul(big.NewInt(6e9), big.NewInt(1e18))) // k
	push(new(big.Int).Mul(big.NewInt(8e8), big.NewInt(1e18))) // dexSupplyThresh
	pushAddr(quoteToken)                                      // quoteTokenAddress
	push(big.NewInt(1))                                       // nativeToQuoteSwapEnabled
	push(big.NewInt(0))                                       // extensionID
	push(big.NewInt(100))                                     // buyTaxRate
	push(big.NewInt(100))                                     // sellTaxRate
	pushAddr(common.Address{})                                // pool
	push(big.NewInt(0))                                       // progress
	push(big.NewInt(0))                                       // lpFeeProfile
	push(big.NewInt(0))                                       // dexId

	return "0x" + common.Bytes2Hex(words)
}

// TestDecodePoolCreated_Success mocks the getTokenV8 RPC call (via a local JSON-RPC server) so the full
// DecodePoolCreated path - including quote token resolution and native-wrapping - is exercised
// deterministically, without depending on live network access.
func TestDecodePoolCreated_Success(t *testing.T) {
	quoteToken := common.HexToAddress("0x0000000000000000000000000000000000000000") // native
	token := common.HexToAddress("0x279c3ef82ed0a886f8fcaa53d7e4215eb9297777")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")

		// ethrpc's single (non-batch) Call() sends one JSON object; Aggregate-style requests send an
		// array. Handle both so this mock works regardless of which the client uses.
		var asArray []map[string]any
		if err := json.Unmarshal(body, &asArray); err == nil {
			resp := make([]map[string]any, 0, len(asArray))
			for _, req := range asArray {
				resp = append(resp, map[string]any{
					"jsonrpc": "2.0",
					"id":      req["id"],
					"result":  encodeTokenStateV8(quoteToken),
				})
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		var asObject map[string]any
		_ = json.Unmarshal(body, &asObject)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      asObject["id"],
			"result":  encodeTokenStateV8(quoteToken),
		})
	}))
	defer srv.Close()

	rpcClient := ethrpc.New(srv.URL)
	config := &Config{
		DexID:         DexType,
		ChainID:       56,
		PortalAddress: "0xe2cE6ab80874Fa9Fa2aAE65D277Dd6B8e65C9De0",
	}
	f := NewPoolFactory(config, rpcClient)

	data := mustPackEvent(t, "TokenCreated",
		big.NewInt(1786013638),
		common.HexToAddress("0x0e13e6a48bf5d49f678459bf2f93ce688c67afa7"),
		big.NewInt(1),
		token,
		"name", "symbol", "meta",
	)

	p, err := f.DecodePoolCreated(ethtypes.Log{
		Topics:      []common.Hash{tokenCreatedEventTopic},
		Data:        data,
		BlockNumber: 12345,
	})
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Equal(t, "0x279c3ef82ed0a886f8fcaa53d7e4215eb9297777", p.Address)
	assert.Equal(t, DexType, p.Type)
	assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
	assert.Equal(t, uint64(12345), p.BlockNumber)
	require.Len(t, p.Tokens, 2)
	// zero-address quote token must be wrapped to the chain's native-wrapped token.
	assert.NotEqual(t, "0x0000000000000000000000000000000000000000", p.Tokens[0].Address)
	assert.Equal(t, "0x279c3ef82ed0a886f8fcaa53d7e4215eb9297777", p.Tokens[1].Address)
}
