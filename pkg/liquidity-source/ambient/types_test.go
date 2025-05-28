package ambient

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestNTokenPool_CanSwapTo(t *testing.T) {
	t.Parallel()
	wethAddr := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	encodedPairs := `[
		"0x0000000000000000000000000000000000000000:0x0f2d719407fdbeff09d87557abb7232601fd9f29",
		"0x0000000000000000000000000000000000000000:0x4e3fbd56cd56c3e72c1403e103b45db9da5b9d2b",
		"0x0000000000000000000000000000000000000000:0x6982508145454ce325ddbe47a25d4ec3d2311933",
		"0x0000000000000000000000000000000000000000:0x6b175474e89094c44da98b954eedeac495271d0f",
		"0x0000000000000000000000000000000000000000:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		"0x0000000000000000000000000000000000000000:0xd533a949740bb3306d119cc777fa900ba034cd52",
		"0x0000000000000000000000000000000000000000:0xdac17f958d2ee523a2206206994597c13d831ec7",
		"0x6b175474e89094c44da98b954eedeac495271d0f:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48:0xdac17f958d2ee523a2206206994597c13d831ec7"
	]`
	var pairs []TokenPair
	err := json.Unmarshal([]byte(encodedPairs), &pairs)
	require.NoError(t, err)

	nPool := NewNTokenPool(pool.Pool{}, pairs, common.HexToAddress(wethAddr))
	require.Equal(t,
		[]string{wethAddr},
		nPool.CanSwapTo("0x0f2d719407fdbeff09d87557abb7232601fd9f29"),
	)
	require.Equal(t,
		[]string{wethAddr},
		nPool.CanSwapTo("0x4e3fbd56cd56c3e72c1403e103b45db9da5b9d2b"),
	)
	require.Equal(t,
		[]string{wethAddr},
		nPool.CanSwapTo("0x6982508145454ce325ddbe47a25d4ec3d2311933"),
	)
	require.Equal(t,
		[]string{wethAddr, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"},
		nPool.CanSwapTo("0x6b175474e89094c44da98b954eedeac495271d0f"),
	)
	require.Equal(t,
		[]string{wethAddr, "0x6b175474e89094c44da98b954eedeac495271d0f", "0xdac17f958d2ee523a2206206994597c13d831ec7"},
		nPool.CanSwapTo("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
	)
	require.Equal(t,
		[]string{wethAddr},
		nPool.CanSwapTo("0xd533a949740bb3306d119cc777fa900ba034cd52"),
	)
	require.Equal(t,
		[]string{wethAddr, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"},
		nPool.CanSwapTo("0xdac17f958d2ee523a2206206994597c13d831ec7"),
	)
	require.Equal(t,
		[]string{
			"0x0f2d719407fdbeff09d87557abb7232601fd9f29",
			"0x4e3fbd56cd56c3e72c1403e103b45db9da5b9d2b",
			"0x6982508145454ce325ddbe47a25d4ec3d2311933",
			"0x6b175474e89094c44da98b954eedeac495271d0f",
			"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			"0xd533a949740bb3306d119cc777fa900ba034cd52",
			"0xdac17f958d2ee523a2206206994597c13d831ec7",
		},
		nPool.CanSwapTo("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"),
	)
}

func TestNTokenPool_GetPair(t *testing.T) {
	t.Parallel()
	wethAddr := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	encodedPairs := `[
		"0x0000000000000000000000000000000000000000:0x0f2d719407fdbeff09d87557abb7232601fd9f29",
		"0x0000000000000000000000000000000000000000:0x4e3fbd56cd56c3e72c1403e103b45db9da5b9d2b",
		"0x0000000000000000000000000000000000000000:0x6982508145454ce325ddbe47a25d4ec3d2311933",
		"0x0000000000000000000000000000000000000000:0x6b175474e89094c44da98b954eedeac495271d0f",
		"0x0000000000000000000000000000000000000000:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		"0x0000000000000000000000000000000000000000:0xd533a949740bb3306d119cc777fa900ba034cd52",
		"0x0000000000000000000000000000000000000000:0xdac17f958d2ee523a2206206994597c13d831ec7",
		"0x6b175474e89094c44da98b954eedeac495271d0f:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48:0xdac17f958d2ee523a2206206994597c13d831ec7"
	]`
	var pairs []TokenPair
	err := json.Unmarshal([]byte(encodedPairs), &pairs)
	require.NoError(t, err)

	nPool := NewNTokenPool(pool.Pool{}, pairs, common.HexToAddress(wethAddr))

	pair, ok := nPool.GetPair(common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7"), common.HexToAddress(wethAddr))
	require.True(t, ok)
	require.Equal(t,
		TokenPair{Base: NativeTokenPlaceholderAddress, Quote: common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7")},
		pair)

	pair, ok = nPool.GetPair(common.HexToAddress(wethAddr), common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"))
	require.True(t, ok)
	require.Equal(t,
		TokenPair{Base: NativeTokenPlaceholderAddress, Quote: common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")},
		pair)

	_, ok = nPool.GetPair(common.HexToAddress("0xdeadbeef"), common.HexToAddress("cafebabe"))
	require.False(t, ok)
}
