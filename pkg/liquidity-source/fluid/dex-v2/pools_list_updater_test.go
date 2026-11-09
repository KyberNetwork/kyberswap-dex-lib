package dexv2

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestPoolFromKey(t *testing.T) {
	d := NewPoolsListUpdater(&Config{DexID: DexType, ChainID: 1, Dex: "0xaBE4D5AA1Bb188f4B5d3a1aE3B8ec5e300A78261"}, nil)
	key := DexKey{Token0: common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"), Token1: common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7"), Fee: big.NewInt(100), TickSpacing: big.NewInt(1)}
	p, err := d.poolFromKey(key, 3)
	require.NoError(t, err)
	require.Equal(t, "0xe018f09af38956affdfeab72c2cefbcd4e6fee44d09df7525ec9dba3e51356a5_3", p.Address)
	require.Equal(t, []string{"0", "0"}, []string(p.Reserves))
	require.Zero(t, p.Tokens[0].Decimals)
	key.Token1 = common.HexToAddress(valueobject.NativeAddress)
	p, err = d.poolFromKey(key, 3)
	require.NoError(t, err)
	require.Equal(t, valueobject.WrapNativeLower(valueobject.NativeAddress, 1), p.Tokens[1].Address)
	var static StaticExtra
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &static))
	require.True(t, static.IsNative[1])
}

func TestLivePermissionedDiscovery(t *testing.T) {
	rpc := os.Getenv("FLUID_DEX_V2_RPC")
	if rpc == "" {
		t.Skip("set FLUID_DEX_V2_RPC")
	}
	c := ethrpc.New(rpc)
	c.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))
	d := NewPoolsListUpdater(&Config{DexID: DexType, ChainID: 1, Dex: "0xabe4d5aa1bb188f4b5d3a1ae3b8ec5e300a78261", Resolver: "0x143c3927eaf0e50a59b1651c2eb1e37fb62aa2a9"}, c)
	pools, metadata, err := d.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools)
	again, _, err := d.GetNewPools(context.Background(), metadata)
	require.NoError(t, err)
	require.Empty(t, again)
}
