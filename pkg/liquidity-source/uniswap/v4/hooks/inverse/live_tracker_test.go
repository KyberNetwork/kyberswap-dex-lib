package inverse

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Opt-in read-only verification of the complete shared tracker -> hook tracker
// -> simulator path. Unlike TestRobinhoodDeployment, this resolves latest via
// the shared tracker rather than supplying HookParam.BlockNumber directly.
func TestRobinhoodCurrentTracker(t *testing.T) {
	rpcURL := os.Getenv("INVERSE_LIVE_RPC")
	if os.Getenv("INVERSE_LIVE_TRACKER") != "true" || rpcURL == "" {
		t.Skip("set INVERSE_LIVE_TRACKER=true and INVERSE_LIVE_RPC for current-pool verification")
	}
	manifest := os.Getenv("INVERSE_LIVE_MANIFEST")
	if manifest == "" {
		manifest = "testdata/deployment.json"
	}
	var deployment struct {
		Plan struct {
			Hook   common.Address `json:"hook"`
			Token  common.Address `json:"token"`
			PoolID common.Hash    `json:"poolId"`
		} `json:"plan"`
	}
	raw, err := os.ReadFile(manifest)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &deployment))
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	multicall := common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11")
	rpcClient := ethrpc.New(rpcURL).SetMulticallContract(multicall)
	client := rpcClient.GetETHClient()
	defer client.Close()
	chain, err := client.ChainID(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(4663), chain.Int64())
	head, err := client.BlockNumber(ctx)
	require.NoError(t, err)
	token0, token1 := deployment.Plan.Token, quoteAddress
	if token0.Cmp(token1) > 0 {
		token0, token1 = token1, token0
	}
	static := uniswapv4.StaticExtra{Fee: 3000, TickSpacing: 60, HooksAddress: deployment.Plan.Hook,
		UniversalRouterAddress: common.HexToAddress("0x8876789976decbfcbbbe364623c63652db8c0904"),
		Permit2Address:         common.HexToAddress("0x000000000022d473030f116ddee9f6b43ac78ba3"), Multicall3Address: multicall}
	staticRaw, err := json.Marshal(static)
	require.NoError(t, err)
	// FetchPoolTicks replaces these two boundary placeholders with actual
	// StateView results at head. The registered hook owns the sole position.
	seedRaw, err := json.Marshal(uniswapv4.Extra{Extra: &uniswapv3.Extra{Ticks: []uniswapv3.Tick{
		{Index: minTick, LiquidityGross: n(0), LiquidityNet: n(0)},
		{Index: maxTick, LiquidityGross: n(0), LiquidityNet: n(0)},
	}}})
	require.NoError(t, err)
	p := entity.Pool{Address: hexutil.Encode(deployment.Plan.PoolID[:]), Type: "uniswap-v4", Exchange: "uniswap-v4",
		BlockNumber: head, SwapFee: 3000, Reserves: entity.PoolReserves{"0", "0"}, StaticExtra: string(staticRaw), Extra: string(seedRaw),
		Tokens: []*entity.PoolToken{{Address: hexutil.Encode(token0[:]), Decimals: 18, Swappable: true}, {Address: hexutil.Encode(token1[:]), Decimals: 18, Swappable: true}}}
	tracker := uniswapv4.NewPoolTracker(&uniswapv4.Config{ChainID: 4663, DexID: "uniswap-v4", StateViewAddress: stateViewAddress}, rpcClient, nil)
	p, err = tracker.FetchPoolTicks(ctx, p)
	require.NoError(t, err)
	p, err = tracker.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
	require.NoError(t, err, "the shared tracker must not pin dependent reads to EVM parent-chain BLOCKNUMBER")
	require.Equal(t, "uniswap-v4-inverse", p.Exchange)
	var extra uniswapv4.Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	var state Extra
	require.NoError(t, json.Unmarshal(extra.HookExtra, &state))
	require.NoError(t, state.validate())
	require.Equal(t, p.BlockNumber, state.BlockNumber)
	simulator, err := uniswapv4.NewPoolSimulator(p, 4663)
	require.NoError(t, err)

	quoterABI, err := abi.JSON(strings.NewReader(`[{"type":"function","name":"quoteExactInputSingle","stateMutability":"nonpayable","inputs":[{"name":"params","type":"tuple","components":[{"name":"poolKey","type":"tuple","components":[{"name":"currency0","type":"address"},{"name":"currency1","type":"address"},{"name":"fee","type":"uint24"},{"name":"tickSpacing","type":"int24"},{"name":"hooks","type":"address"}]},{"name":"zeroForOne","type":"bool"},{"name":"exactAmount","type":"uint128"},{"name":"hookData","type":"bytes"}]}],"outputs":[{"name":"amountOut","type":"uint256"},{"name":"gasEstimate","type":"uint256"}]}]`))
	require.NoError(t, err)
	quoter := common.HexToAddress("0x8dc178efb8111bb0973dd9d722ebeff267c98f94")
	type poolKey struct {
		Currency0   common.Address
		Currency1   common.Address
		Fee         *big.Int
		TickSpacing *big.Int
		Hooks       common.Address
	}
	type quoterParams struct {
		PoolKey     poolKey
		ZeroForOne  bool
		ExactAmount *big.Int
		HookData    []byte
	}
	for _, tc := range []struct{ name, input, output, amount string }{
		{"buy", hexutil.Encode(quoteAddress[:]), hexutil.Encode(deployment.Plan.Token[:]), "10000000000000"},
		{"sell", hexutil.Encode(deployment.Plan.Token[:]), hexutil.Encode(quoteAddress[:]), "1000000000000000000000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			amount, ok := new(big.Int).SetString(tc.amount, 10)
			require.True(t, ok)
			quoted, e := simulator.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tc.input, Amount: amount}, TokenOut: tc.output})
			require.NoError(t, e)
			data, e := quoterABI.Pack("quoteExactInputSingle", quoterParams{
				PoolKey:    poolKey{token0, token1, n(3000), n(60), deployment.Plan.Hook},
				ZeroForOne: common.HexToAddress(tc.input) == token0, ExactAmount: amount, HookData: []byte{},
			})
			require.NoError(t, e)
			result, e := client.CallContract(ctx, ethereum.CallMsg{To: &quoter, Data: data}, new(big.Int).SetUint64(p.BlockNumber))
			require.NoError(t, e)
			values, e := quoterABI.Unpack("quoteExactInputSingle", result)
			require.NoError(t, e)
			require.Equal(t, values[0].(*big.Int).String(), quoted.TokenAmountOut.Amount.String())
			t.Logf("RPC block %d: input=%s; Kyber=EVM output=%s", p.BlockNumber, amount, quoted.TokenAmountOut.Amount)
		})
	}
}
