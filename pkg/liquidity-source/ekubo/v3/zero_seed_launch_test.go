package ekubov3

import (
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/abis"
	ekubomath "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestZeroSeedLaunchSnapshotAndSimulator(t *testing.T) {
	for _, reorg := range []bool{false, true} {
		t.Run(map[bool]string{false: "pinned", true: "reorg"}[reorg], func(t *testing.T) {
			ext := common.HexToAddress("0x517621Ea1bf51F31826819b118fa4dfEb4D198D2")
			key := anyPoolKey(valueobject.ZeroAddress, "0x0000000000000000000000000000000000000100", ext.Hex(), 0, pools.NewConcentratedPoolTypeConfig(17990))
			launch := pools.Launch{Controller: common.HexToAddress("0x123"), TradingStart: 1000, DecayRateX32: 68913100933, InitialFee: 922337203685477580, FinalFee: 55340232221128654, TickLower: 9210880, TickUpper: 13798330}
			encodedLaunch, err := abis.ZeroSeedLaunchABI.Methods["getLaunch"].Outputs.Pack(launch)
			require.NoError(t, err)
			require.Equal(t, "0x7d1d589c", hexutil.Encode(abis.ZeroSeedLaunchABI.Methods["getLaunch"].ID))
			liquidity, _ := new(big.Int).SetString("1121810267768648385525007", 10)
			data := []struct {
				Tick      int32
				SqrtRatio *big.Int
				Liquidity *big.Int
				MinTick   int32
				MaxTick   int32
				Ticks     []pools.TickRPC
			}{{13798330, ekubomath.FixedSqrtRatioToFloat(ekubomath.ToSqrtRatio(13798330), false).ToBig(), big.NewInt(0), 9210880, 13798330, []pools.TickRPC{{Number: 9210880, LiquidityDelta: liquidity}, {Number: 13798330, LiquidityDelta: new(big.Int).Neg(liquidity)}}}}
			encodedData, err := abis.QuoteDataFetcherABI.Methods[quoteDataFetcherMethod].Outputs.Pack(data)
			require.NoError(t, err)
			u256, _ := abi.NewType("uint256", "", nil)
			bytesArray, _ := abi.NewType("bytes[]", "", nil)
			response, err := (abi.Arguments{{Type: u256}, {Type: bytesArray}}).Pack(big.NewInt(123), [][]byte{encodedLaunch, encodedData})
			require.NoError(t, err)
			headers, calls := 0, 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					ID     json.RawMessage   `json:"id"`
					Method string            `json:"method"`
					Params []json.RawMessage `json:"params"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					http.Error(w, "bad request", 400)
					return
				}
				var result any
				switch req.Method {
				case "eth_getBlockByNumber":
					headers++
					header := &types.Header{Number: big.NewInt(123), Difficulty: big.NewInt(0), Time: 1000}
					if headers > 1 && reorg {
						header.Extra = []byte{1}
					}
					result = header
				case "eth_call":
					calls++
					if string(req.Params[1]) != `"0x7b"` {
						t.Errorf("unpinned state call: %s", req.Params[1])
					}
					if !strings.Contains(string(req.Params[0]), "7d1d589c") {
						t.Error("missing getLaunch call")
					}
					result = hexutil.Encode(response)
				default:
					t.Errorf("unexpected RPC %s", req.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
			}))
			defer srv.Close()
			cfg := &Config{ChainId: valueobject.ChainIDRobinhood, ZeroSeedLaunch: ext, Core: common.HexToAddress("0x14"), QuoteDataFetcher: common.HexToAddress("0x15").Hex()}
			fetcher := NewDataFetchers(ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0x16")), cfg)
			fetched, err := fetcher.fetchPools(t.Context(), []pools.AnyPoolKey{key}, nil)
			require.Equal(t, 1, calls)
			require.Equal(t, 2, headers)
			if reorg {
				require.ErrorIs(t, err, ErrReorg)
				return
			}
			require.NoError(t, err)
			require.Len(t, fetched, 1)
			require.Equal(t, uint64(123), fetched[0].blockNumber)
			state := fetched[0].GetState().(*pools.ZeroSeedLaunchPoolState)
			require.Equal(t, uint64(1000), state.BlockTimestamp)
			extra, err := json.Marshal(state)
			require.NoError(t, err)
			static, err := json.Marshal(StaticExtra{Core: cfg.Core, ExtensionType: ExtensionTypeZeroSeedLaunch, PoolKey: key})
			require.NoError(t, err)
			sim, err := NewPoolSimulator(entity.Pool{Extra: string(extra), StaticExtra: string(static), BlockNumber: 123, Tokens: []*entity.PoolToken{{Address: "quote"}, {Address: "launch"}}, Reserves: []string{"0", "0"}})
			require.NoError(t, err)
			q, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: "quote", Amount: new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)}, TokenOut: "launch"})
			require.NoError(t, err)
			require.Equal(t, "932986871256060603067126", q.TokenAmountOut.Amount.String())
			require.Equal(t, "launch", q.Fee.Token)
			require.Equal(t, "49104572171371610642911", q.Fee.Amount.String())
			require.Equal(t, uint64(123), sim.GetMetaInfo("quote", "launch").(Meta).BlockNumber)
			sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: q.SwapInfo})
			buy, err := sim.CalcAmountIn(pool.CalcAmountInParams{TokenAmountOut: pool.TokenAmount{Token: "launch", Amount: q.TokenAmountOut.Amount}, TokenIn: "quote"})
			require.NoError(t, err)
			require.Equal(t, "1001724835881546317", buy.TokenAmountIn.Amount.String())
			require.Equal(t, "quote", buy.Fee.Token)
		})
	}
}

func TestZeroSeedLaunchDiscoveryFilter(t *testing.T) {
	ext := common.HexToAddress("0x517621Ea1bf51F31826819b118fa4dfEb4D198D2")
	cfg := &Config{ChainId: valueobject.ChainIDRobinhood, ZeroSeedLaunch: ext}
	require.Equal(t, ExtensionTypeUnknown, (&Config{ChainId: valueobject.ChainIDEthereum, ZeroSeedLaunch: ext}).ExtensionType(ext))
	require.Equal(t, ExtensionTypeUnknown, (&Config{ChainId: valueobject.ChainIDRobinhood, ZeroSeedLaunch: common.HexToAddress("0x5100000000000000000000000000000000000000")}).ExtensionType(common.HexToAddress("0x5100000000000000000000000000000000000000")))
	require.Equal(t, ExtensionTypeZeroSeedLaunch, cfg.ExtensionType(ext))
	require.Equal(t, ExtensionTypeUnknown, cfg.ExtensionType(common.HexToAddress("0x517621Ea1bf51F31826819b118fa4dfEb4D198D3")))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Variables struct {
				Extensions []common.Address `json:"extensions"`
			} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		require.Contains(t, req.Variables.Extensions, ext)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"poolInitializations":[]}}`))
	}))
	defer srv.Close()
	u := NewPoolListUpdater(cfg, ethrpc.New(testFailingRPCURL), graphql.NewClient(srv.URL))
	keys, _, err := u.getNewPoolKeys(t.Context())
	require.NoError(t, err)
	require.Empty(t, keys)
}
