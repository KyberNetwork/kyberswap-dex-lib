package evplusai

import (
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	livePool      = "0xc7b615a3721594f73664eb3f62d8290d0fcc8d9d1156aed1ddecbd7f32efd9f5"
	liveStateView = "0xf3334192d15450cdd385c8b70e03f9a6bd9e673b"
	liveQuoter    = "0x8dc178efb8111bb0973dd9d722ebeff267c98f94"
	usdg          = "0x5fc5360d0400a0fd4f2af552add042d716f1d168"
	weth          = "0x0bd7d308f8e1639fab988df18a8011f41eacad73"
)

type fixture struct {
	Pool   entity.Pool `json:"pool"`
	Quotes []quoteCase `json:"quotes"`
}
type quoteCase struct {
	Name       string          `json:"name"`
	Extra      json.RawMessage `json:"hookExtra"`
	Sell       bool            `json:"sell"`
	ExactInput bool            `json:"exactInput"`
	Amount     string          `json:"amount"`
	Expected   string          `json:"expected"`
	Gas        uint64          `json:"quoterGas"`
}

func compareQuote(t *testing.T, f fixture, q quoteCase) {
	t.Helper()
	p := f.Pool
	var extra uniswapv4.Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	extra.HookExtra = q.Extra
	raw, err := json.Marshal(extra)
	require.NoError(t, err)
	p.Extra = string(raw)
	sim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDRobinhood)
	require.NoError(t, err)
	input, output := usdg, weth
	if q.Sell {
		input, output = weth, usdg
	}
	amount, ok := new(big.Int).SetString(q.Amount, 10)
	require.True(t, ok)
	// Repeated quotes and cloned state must give the same result.
	for i := 0; i < 2; i++ {
		if i == 1 {
			sim = sim.CloneState().(*uniswapv4.PoolSimulator)
		}
		if q.ExactInput {
			result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: input, Amount: amount}, TokenOut: output})
			require.NoError(t, err)
			require.Equal(t, q.Expected, result.TokenAmountOut.Amount.String(), q.Name)
		} else {
			result, err := sim.CalcAmountIn(pool.CalcAmountInParams{TokenAmountOut: pool.TokenAmount{Token: output, Amount: amount}, TokenIn: input})
			require.NoError(t, err)
			require.Equal(t, q.Expected, result.TokenAmountIn.Amount.String(), q.Name)
		}
	}
}

// Explicit opt-in, read-only eth_calls. State overrides exercise the actual
// deployed bytecode at different fees without sending transactions or using keys.
// EVPLUSAI_WRITE_FIXTURE writes the deterministic regression fixture after success.
func TestRPCQuotes(t *testing.T) {
	url := os.Getenv("EVPLUSAI_RPC_URL")
	if url == "" {
		t.Skip("set EVPLUSAI_RPC_URL to run deployed-contract comparisons")
	}
	client := ethrpc.New(url).SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))
	block, err := client.GetBlockNumber(t.Context())
	require.NoError(t, err)
	if b := os.Getenv("EVPLUSAI_BLOCK"); b != "" {
		block, err = strconv.ParseUint(b, 10, 64)
		require.NoError(t, err)
	}
	bn := new(big.Int).SetUint64(block)
	header, err := client.GetETHClient().HeaderByNumber(t.Context(), bn)
	require.NoError(t, err)
	id := common.HexToHash(livePool)
	viewABI := parseABI(`[
 {"type":"function","name":"getLiquidity","inputs":[{"type":"bytes32"}],"outputs":[{"type":"uint128"}]},
 {"type":"function","name":"getTickBitmap","inputs":[{"type":"bytes32"},{"type":"int16"}],"outputs":[{"type":"uint256"}]},
 {"type":"function","name":"getTickLiquidity","inputs":[{"type":"bytes32"},{"type":"int24"}],"outputs":[{"name":"liquidityGross","type":"uint128"},{"name":"liquidityNet","type":"int128"}]}
 ]`)
	var slot slot0
	var liquidity *big.Int
	_, err = client.NewRequest().SetContext(t.Context()).SetBlockNumber(bn).
		AddCall(&ethrpc.Call{ABI: stateViewABI, Target: liveStateView, Method: "getSlot0", Params: []any{id}}, []any{&slot}).
		AddCall(&ethrpc.Call{ABI: viewABI, Target: liveStateView, Method: "getLiquidity", Params: []any{id}}, []any{&liquidity}).Aggregate()
	require.NoError(t, err)
	// Read every bitmap word for tick spacing 10, in bounded RPC batches.
	var ticks []uniswapv3.Tick
	for start := -347; start <= 346; start += 100 {
		end := min(start+100, 347)
		words := make([]*big.Int, end-start)
		req := client.NewRequest().SetContext(t.Context()).SetBlockNumber(bn)
		for word := start; word < end; word++ {
			req.AddCall(&ethrpc.Call{ABI: viewABI, Target: liveStateView, Method: "getTickBitmap", Params: []any{id, int16(word)}}, []any{&words[word-start]})
		}
		_, err = req.Aggregate()
		require.NoError(t, err)
		for i, bits := range words {
			for bit := 0; bit < 256; bit++ {
				if bits.Bit(bit) != 0 {
					ticks = append(ticks, uniswapv3.Tick{Index: ((start+i)*256 + bit) * 10})
				}
			}
		}
	}
	require.NotEmpty(t, ticks)
	req := client.NewRequest().SetContext(t.Context()).SetBlockNumber(bn)
	for i := range ticks {
		req.AddCall(&ethrpc.Call{ABI: viewABI, Target: liveStateView, Method: "getTickLiquidity", Params: []any{id, big.NewInt(int64(ticks[i].Index))}}, []any{&ticks[i]})
	}
	_, err = req.Aggregate()
	require.NoError(t, err)
	ext, err := json.Marshal(uniswapv4.Extra{Extra: &uniswapv3.Extra{Liquidity: liquidity, SqrtPriceX96: slot.SqrtPriceX96, Tick: slot.Tick, TickSpacing: 10, Ticks: ticks}})
	require.NoError(t, err)
	static, err := json.Marshal(uniswapv4.StaticExtra{IsNative: [2]bool{true, false}, Fee: 8388608, TickSpacing: 10, HooksAddress: HookAddress})
	require.NoError(t, err)
	reserve0, reserve1 := uniswapv4.EstimateReservesFromTicks(slot.SqrtPriceX96, ticks)
	f := fixture{Pool: entity.Pool{Address: livePool, Type: uniswapv4.DexType, Exchange: valueobject.ExchangeUniswapV4EVPLUSAI,
		BlockNumber: block, Timestamp: int64(header.Time), SwapFee: 500, Reserves: []string{reserve0.String(), reserve1.String()},
		Tokens: []*entity.PoolToken{{Address: weth, Decimals: 18, Swappable: true}, {Address: usdg, Decimals: 6, Swappable: true}}, Extra: string(ext), StaticExtra: string(static)}}
	quoterABI := parseABI(`[` + quoteABIMethod("quoteExactInputSingle") + `,` + quoteABIMethod("quoteExactOutputSingle") + `]`)
	type poolKey struct {
		Currency0   common.Address
		Currency1   common.Address
		Fee         *big.Int
		TickSpacing *big.Int
		Hooks       common.Address
	}
	type quoteParams struct {
		PoolKey     poolKey
		ZeroForOne  bool
		ExactAmount *big.Int
		HookData    []byte
	}
	key := poolKey{Currency1: common.HexToAddress(usdg), Fee: big.NewInt(8388608), TickSpacing: big.NewInt(10), Hooks: HookAddress}
	// _fees is mapping slot 5 in the verified EVPLUSAI storage layout.
	storageKey := crypto.Keccak256Hash(id.Bytes(), common.LeftPadBytes([]byte{5}, 32))
	for _, scenario := range []struct {
		name    string
		f0, f1  uint64
		expired bool
	}{
		{"live", 0, 0, false}, {"min-max", 250, 9000, false}, {"max-min", 9000, 250, false},
		{"odd", 7919, 837, false}, {"expiry-boundary", 9000, 250, true},
	} {
		var overrides map[common.Address]gethclient.OverrideAccount
		if scenario.name != "live" {
			expiry := header.Time + 300
			if scenario.expired {
				expiry = header.Time
			}
			packed := new(big.Int).SetUint64(scenario.f0)
			packed.Or(packed, new(big.Int).Lsh(new(big.Int).SetUint64(scenario.f1), 24))
			packed.Or(packed, new(big.Int).Lsh(new(big.Int).SetUint64(header.Time), 48))
			packed.Or(packed, new(big.Int).Lsh(new(big.Int).SetUint64(expiry), 112))
			overrides = map[common.Address]gethclient.OverrideAccount{HookAddress: {StateDiff: map[common.Hash]common.Hash{storageKey: common.BigToHash(packed)}}}
		}
		h := &Hook{}
		tracked, err := h.Track(t.Context(), &uniswapv4.HookParam{Cfg: &uniswapv4.Config{ChainID: 4663, StateViewAddress: liveStateView}, RpcClient: client, Pool: &f.Pool, HookAddress: HookAddress, BlockNumber: bn, Overrides: overrides})
		require.NoError(t, err)
		var snapshot Extra
		require.NoError(t, json.Unmarshal(tracked, &snapshot))
		require.Equal(t, header.Time, snapshot.Timestamp)
		if scenario.expired {
			fee, err := snapshot.budget(true)
			require.NoError(t, err)
			require.EqualValues(t, 500, fee)
		}
		for _, sell := range []bool{true, false} {
			for _, exactIn := range []bool{true, false} {
				for _, large := range []bool{false, true} {
					amount := big.NewInt(1000000) // one USDG
					if sell == exactIn {
						amount = big.NewInt(1000000000000000)
					} // 0.001 ETH
					if large {
						amount.Mul(amount, big.NewInt(100))
					}
					method := "quoteExactOutputSingle"
					if exactIn {
						method = "quoteExactInputSingle"
					}
					var quoted struct {
						Amount      *big.Int
						GasEstimate *big.Int
					}
					_, err = client.NewRequest().SetContext(t.Context()).SetBlockNumber(bn).SetOverrides(overrides).
						AddCall(&ethrpc.Call{ABI: quoterABI, Target: liveQuoter, Method: method, Params: []any{quoteParams{key, sell, amount, []byte{}}}}, []any{&quoted}).Call()
					require.NoError(t, err)
					q := quoteCase{Name: fmt.Sprintf("%s/sell=%t/exactIn=%t/large=%t", scenario.name, sell, exactIn, large), Extra: tracked, Sell: sell, ExactInput: exactIn, Amount: amount.String(), Expected: quoted.Amount.String(), Gas: quoted.GasEstimate.Uint64()}
					compareQuote(t, f, q)
					f.Quotes = append(f.Quotes, q)
					t.Logf("block=%d %s amount=%s expected=%s gas=%d", block, q.Name, q.Amount, q.Expected, q.Gas)
				}
			}
		}
	}
	if path := os.Getenv("EVPLUSAI_WRITE_FIXTURE"); path != "" {
		data, err := json.MarshalIndent(f, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, append(data, '\n'), 0644))
	}
}

func quoteABIMethod(name string) string {
	return strings.ReplaceAll(`{"type":"function","name":"METHOD","inputs":[{"name":"params","type":"tuple","components":[{"name":"poolKey","type":"tuple","components":[{"name":"currency0","type":"address"},{"name":"currency1","type":"address"},{"name":"fee","type":"uint24"},{"name":"tickSpacing","type":"int24"},{"name":"hooks","type":"address"}]},{"name":"zeroForOne","type":"bool"},{"name":"exactAmount","type":"uint128"},{"name":"hookData","type":"bytes"}]}],"outputs":[{"name":"amount","type":"uint256"},{"name":"gasEstimate","type":"uint256"}]}`, "METHOD", name)
}
