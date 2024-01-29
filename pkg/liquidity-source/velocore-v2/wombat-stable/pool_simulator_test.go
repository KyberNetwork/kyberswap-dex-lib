package wombatstable

import (
	"fmt"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOutConcurrentSafe(t *testing.T) {
	type testcase struct {
		name        string
		poolEncoded string
		tokenIn     string
		amountIn    string
		tokenOut    string
	}
	testcases := []testcase{
		{
			name: "swap BUSD for USDC.e",
			poolEncoded: `{
				"address": "0x61cb3a0c59825464474ebb287a3e7d2b9b59d093",
				"type": "velocore-v2-wombat-stable",
				"timestamp": 1705576647,
				"reserves": [
					"11195773019488324321309",
					"9192257736"
				],
				"tokens": [
					{
						"address": "0x7d43aabc515c356145049227cee54b608342c0ad",
						"weight": 1,
						"swappable": true
					},
					{
						"address": "0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
						"weight": 1,
						"swappable": true
					}
				],
				"extra": "{\"amp\":250000000000000,\"fee1e18\":100000000000000,\"lpTokenBalances\":{\"0x176211869ca2b568f2a7d4ee941e073a821ee1ff\":340282366920938463463374607416268936368,\"0x7d43aabc515c356145049227cee54b608342c0ad\":340282366920938458576602139746458171455},\"tokenInfo\":{\"0x176211869ca2b568f2a7d4ee941e073a821ee1ff\":{\"indexPlus1\":2,\"scale\":12},\"0x7d43aabc515c356145049227cee54b608342c0ad\":{\"indexPlus1\":1,\"scale\":0}}}",
				"staticExtra": "{\"vault\":\"0x1d0188c4B276A09366D05d6Be06aF61a73bC7535\",\"wrappers\":{\"0x1e1f509963a6d33e169d9497b11c7dbfe73b7f13\":\"0xb30e7a2e6f7389ca5ddc714da4c991b7a1dcc88e\",\"0xb79dd08ea68a908a97220c76d19a6aa9cbde4376\":\"0x3f006b0493ff32b33be2809367f5f6722cb84a7b\"}}",
				"blockNumber": 1711060
			}`,
			tokenIn:  "0x7d43aabc515c356145049227cee54b608342c0ad",
			amountIn: "1000000000000000000000", // 1000 BUSD
			tokenOut: "0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
		},
	}
	for _, tc := range testcases {
		poolEntity := new(entity.Pool)
		err := json.Unmarshal([]byte(tc.poolEncoded), poolEntity)
		require.NoError(t, err)

		poolSim, err := NewPoolSimulator(*poolEntity)
		require.NoError(t, err)

		result, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
			return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  tc.tokenIn,
					Amount: bignumber.NewBig10(tc.amountIn),
				},
				TokenOut: tc.tokenOut,
			})
		})
		require.NoError(t, err)
		fmt.Printf("%s\n", result.TokenAmountOut.Amount)
		_ = result
	}
}

// func getPoolEntity(poolAddr string) (string, error) {
// 	cfg := &Config{
// 		VaultAddress:    "0x1d0188c4B276A09366D05d6Be06aF61a73bC7535",
// 		RegistryAddress: "0x111A6d7f5dDb85776F1b6A6DEAbe552815559f9E",
// 		NewPoolLimit:    100,
// 		LensAddress:     "0xaA18cDb16a4DD88a59f4c2f45b5c91d009549e06",
// 		Wrappers: map[string]string{
// 			"0x1e1f509963a6d33e169d9497b11c7dbfe73b7f13": "0xb30e7a2e6f7389ca5ddc714da4c991b7a1dcc88e",
// 			"0xb79dd08ea68a908a97220c76d19a6aa9cbde4376": "0x3f006b0493ff32b33be2809367f5f6722cb84a7b",
// 		},
// 	}

// 	ethrpcClient := ethrpc.New("https://rpc.linea.build")
// 	ethrpcClient.SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

// 	poolsListUpdater := NewPoolsListUpdater(cfg, ethrpcClient)
// 	pools, _, err := poolsListUpdater.GetNewPools(context.TODO(), nil)
// 	if err != nil {
// 		return "", err
// 	}

// 	var _p *entity.Pool
// 	for _, p := range pools {
// 		if strings.EqualFold(p.Address, poolAddr) {
// 			_p = &p
// 			break
// 		}
// 	}

// 	if _p == nil {
// 		return "", fmt.Errorf("pool %s not found", poolAddr)
// 	}

// 	poolTracker, err := NewPoolTracker(cfg, ethrpcClient)
// 	if err != nil {
// 		return "", err
// 	}
// 	poolEntity, err := poolTracker.GetNewPoolState(context.TODO(), *_p, pool.GetNewPoolStateParams{})
// 	if err != nil {
// 		return "", err
// 	}

// 	poolEncoded, _ := json.MarshalIndent(poolEntity, "", "\t")

// 	return string(poolEncoded), nil
// }

// func TestGetPoolEntity(t *testing.T) {
// 	poolEncoded, err := getPoolEntity("0x61cb3a0c59825464474ebb287a3e7d2b9b59d093")
// 	require.NoError(t, err)
// 	fmt.Printf("%s\n", poolEncoded)
// }
