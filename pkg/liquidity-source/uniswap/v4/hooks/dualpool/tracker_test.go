package dualpool

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// Live read of the Twofold NVDA/USDG pool on Robinhood Chain through the hook's
// Track: exercises the PoolKey tuple packing, the distribution tuple[] decoding
// and the StateView slot0 read against the real contracts.
//
// The public RPC keeps only recent state, so the test reads at a block just behind
// head and checks Track against direct reads at that same block and against
// properties that hold at any block, not values frozen on one date.
func TestHook_Track_Live(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip()
	}
	rpc := os.Getenv("ROBINHOOD_RPC")
	if rpc == "" {
		rpc = "https://rpc.mainnet.chain.robinhood.com"
	}
	client := ethrpc.New(rpc).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	head, err := client.GetBlockNumber(context.Background())
	require.NoError(t, err)
	block := new(big.Int).SetUint64(head - 5)

	p := entity.Pool{
		Address: "0x085e812d2b072f9569c192769fef074eac9b3b519647665c98a7e4119f8fa06b",
		Tokens: []*entity.PoolToken{
			{Address: "0x5fc5360d0400a0fd4f2af552add042d716f1d168", Decimals: 6, Swappable: true},
			{Address: "0xd0601ce157db5bdc3162bbac2a2c8af5320d9eec", Decimals: 18, Swappable: true},
		},
		StaticExtra: `{"0x0":[false,false],"fee":500,"tS":10,"hooks":"0xd1bcbcca41f3bdb6b4812652959c6df725ea2ac0"}`,
	}
	hookAddr := HookAddresses[0]
	param := &uniswapv4.HookParam{
		Cfg:         &uniswapv4.Config{StateViewAddress: defaultStateViewRobinhood},
		RpcClient:   client,
		Pool:        &p,
		HookAddress: hookAddr,
		BlockNumber: block,
	}
	h, ok := uniswapv4.GetHook(hookAddr, param)
	require.True(t, ok)

	raw, err := h.Track(context.Background(), param)
	require.NoError(t, err)
	var extra Extra
	require.NoError(t, json.Unmarshal(raw, &extra))

	// Direct reads at the same block, one call each, as the reference.
	var (
		bal  effectiveLiquidityResp
		live bool
		s0   slot0Resp
	)
	key := uniswapv4.PoolKey{
		Currency0:   common.HexToAddress(p.Tokens[0].Address),
		Currency1:   common.HexToAddress(p.Tokens[1].Address),
		Fee:         big.NewInt(500),
		TickSpacing: big.NewInt(10),
		Hooks:       common.HexToAddress("0xd1bcbcca41f3bdb6b4812652959c6df725ea2ac0"),
	}
	hook := hexutil.Encode(hookAddr[:])
	poolID := common.HexToHash(p.Address)
	_, err = client.NewRequest().SetBlockNumber(block).
		AddCall(&ethrpc.Call{ABI: dualPoolHookABI, Target: hook, Method: "getEffectiveLiquidity", Params: []any{key}}, []any{&bal}).
		AddCall(&ethrpc.Call{ABI: dualPoolHookABI, Target: hook, Method: "livePools", Params: []any{poolID}}, []any{&live}).
		AddCall(&ethrpc.Call{ABI: stateViewABI, Target: defaultStateViewRobinhood, Method: "getSlot0", Params: []any{poolID}}, []any{&s0}).
		Aggregate()
	require.NoError(t, err)

	require.Equal(t, live, extra.Live)
	require.Equal(t, bal.Token0.String(), extra.Balance0.Dec())
	require.Equal(t, bal.Token1.String(), extra.Balance1.Dec())
	require.Equal(t, s0.SqrtPriceX96.String(), extra.SqrtPriceX96.Dec())
	require.Equal(t, int32(s0.Tick.Int64()), extra.Tick)
	require.Equal(t, uint32(s0.LpFee.Uint64()), extra.LpFee)

	// Properties of the pool at any block: static 500 fee, tick matches the price,
	// and the distribution splits the reserves into ordered ranges summing to 100%.
	require.Equal(t, uint32(500), extra.LpFee)
	wantTick, err := uniswapv3.GetTickAtSqrtRatio(extra.SqrtPriceX96)
	require.NoError(t, err)
	require.Equal(t, wantTick, int(extra.Tick))
	require.NotEmpty(t, extra.Buckets)
	var weight uint32
	for _, b := range extra.Buckets {
		require.Less(t, b.TickLower, b.TickUpper)
		weight += uint32(b.WeightBps)
	}
	require.Equal(t, uint32(10_000), weight)

	reserves, err := h.GetReserves(context.Background(), param)
	require.NoError(t, err)
	require.Equal(t, entity.PoolReserves{extra.Balance0.Dec(), extra.Balance1.Dec()}, reserves)
}
