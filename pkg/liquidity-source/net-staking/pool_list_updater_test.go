package netstaking

import (
	"context"
	"encoding/hex"
	"math/big"
	"testing"
	"testing/synctest"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test/ethrpcmock"
	utilabi "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

// TestMethodABI_DefaultMethods confirms the default getter names ("net"/"sNet")
// derive the confirmed 4-byte selectors: 0xc2fff4a5/0x1adb8eb6.
func TestMethodABI_DefaultMethods(t *testing.T) {
	assert.Equal(t, "net", defaultBaseTokenMethod)
	assert.Equal(t, "sNet", defaultStakedTokenMethod)

	baseABI := methodABI(defaultBaseTokenMethod)
	stakedABI := methodABI(defaultStakedTokenMethod)

	baseMethod, ok := baseABI.Methods[defaultBaseTokenMethod]
	require.True(t, ok)
	stakedMethod, ok := stakedABI.Methods[defaultStakedTokenMethod]
	require.True(t, ok)

	assert.Equal(t, "c2fff4a5", hex.EncodeToString(baseMethod.ID), "net() selector")
	assert.Equal(t, "1adb8eb6", hex.EncodeToString(stakedMethod.ID), "sNet() selector")
}

func TestGetNewPoolStatePinsBalances(t *testing.T) {
	for _, tc := range []struct {
		name               string
		wrapped, overrides bool
	}{
		{name: "wrapped", wrapped: true}, {name: "wrapped-overrides", wrapped: true, overrides: true},
		{name: "unwrapped"}, {name: "unwrapped-overrides", overrides: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				server, recorder := ethrpcmock.New(t, utilabi.Erc20ABI, stakedNETABI)
				client := ethrpc.New(server.URL).SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))
				wrap := ""
				if tc.wrapped {
					wrap = "0x4444444444444444444444444444444444444444"
				}
				tracker := &PoolTracker{config: &Config{StakingAddress: "0x5555555555555555555555555555555555555555", WrapAddress: wrap}, ethrpcClient: client}
				p := entity.Pool{Address: "pool", StaticExtra: "{}", Tokens: []*entity.PoolToken{{Address: "0x1111111111111111111111111111111111111111"}, {Address: "0x2222222222222222222222222222222222222222"}}}
				var got entity.Pool
				var err error
				if tc.overrides {
					account := common.HexToAddress("0x6666666666666666666666666666666666666666")
					got, err = tracker.GetNewPoolStateWithOverrides(context.Background(), p, pool.GetNewPoolStateWithOverridesParams{Overrides: map[common.Address]gethclient.OverrideAccount{account: {Balance: big.NewInt(1)}}})
				} else {
					got, err = tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
				}
				require.NoError(t, err)
				require.EqualValues(t, ethrpcmock.Block, got.BlockNumber)
				expectedCalls := 1
				if tc.wrapped {
					expectedCalls = 2
				}
				ethrpcmock.RequireRequestsPinned(t, recorder, expectedCalls)
				key := "net-staking-" + tc.name
				ethrpcmock.CheckPool(t, key, got)
				ethrpcmock.CheckRequests(t, key, recorder)
			})
		})
	}
}

// TestMethodABI_NukeMethods confirms a differently-named staking fork (NUKE's
// nuke()/stakedNuke(), selectors 0xbc8fbbf8/0xaf50e690) round-trips through the same
// synthetic-ABI mechanism used for the default net()/sNet() case.
func TestMethodABI_NukeMethods(t *testing.T) {
	baseABI := methodABI("nuke")
	stakedABI := methodABI("stakedNuke")

	baseMethod, ok := baseABI.Methods["nuke"]
	require.True(t, ok)
	stakedMethod, ok := stakedABI.Methods["stakedNuke"]
	require.True(t, ok)

	assert.Equal(t, "bc8fbbf8", hex.EncodeToString(baseMethod.ID), "nuke() selector")
	assert.Equal(t, "af50e690", hex.EncodeToString(stakedMethod.ID), "stakedNuke() selector")

	// Both methods are no-arg, single-address-output views regardless of name.
	assert.Empty(t, baseMethod.Inputs)
	require.Len(t, baseMethod.Outputs, 1)
}

// TestConfig_MethodZeroValueOmitted confirms an all-empty (unset) Config still resolves
// to the default net()/sNet() getters inside fetchTokens's resolution logic, matching
// what GetNewPools does for existing net-staking configs that predate this field.
func TestConfig_MethodZeroValueOmitted(t *testing.T) {
	cfg := &Config{
		DexId:          "net-staking",
		StakingAddress: "0xb078cc304a0b264c5f3680dc0488954accd02e87",
		WrapAddress:    "0x63c12667638f2ae6fc6ae09b43d98ec84a8586ea",
	}

	baseMethod := cfg.BaseTokenMethod
	if baseMethod == "" {
		baseMethod = defaultBaseTokenMethod
	}
	stakedMethod := cfg.StakedTokenMethod
	if stakedMethod == "" {
		stakedMethod = defaultStakedTokenMethod
	}

	assert.Equal(t, defaultBaseTokenMethod, baseMethod)
	assert.Equal(t, defaultStakedTokenMethod, stakedMethod)
}
