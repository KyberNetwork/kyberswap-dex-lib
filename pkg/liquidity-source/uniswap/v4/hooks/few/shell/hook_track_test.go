package fewshell

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// chain is the on-chain state the stubbed RPC answers from.
type chain struct {
	pairToken0         string
	reserve0, reserve1 string
	paused             bool
	// balances[token][holder]
	balances map[string]map[string]string
}

var (
	poolManager = uniswapv4.PoolManager[valueobject.ChainIDEthereum]
	otherHolder = common.HexToAddress("0x1111111111111111111111111111111111111111")
)

func liveChain() *chain {
	return &chain{
		pairToken0: fwWETH, reserve0: "16929650201529705921438", reserve1: "29261029832819",
		balances: map[string]map[string]string{
			weth: {poolManager.Hex(): "564000000000000000000", fwWETH: "1833000000000000000000", otherHolder.Hex(): "7"},
			usdt: {poolManager.Hex(): "72000000000000", fwUSDT: "1978000000000000", otherHolder.Hex(): "7"},
		},
	}
}

func (c *chain) respond(t testing.TB) func(common.Address, []byte) (bool, []byte) {
	return func(target common.Address, data []byte) (bool, []byte) {
		method, err := shellABI.MethodById(data[:4])
		if err != nil {
			method, err = abi.Erc20ABI.MethodById(data[:4])
		}
		require.NoError(t, err)
		pack := func(out ...any) (bool, []byte) {
			ret, err := method.Outputs.Pack(out...)
			require.NoError(t, err)
			return true, ret
		}
		switch method.Name {
		case "registeredRouteFor":
			assert.Equal(t, HookAddresses[0], target)
			args, err := method.Inputs.Unpack(data[4:])
			require.NoError(t, err)
			assert.Equal(t, common.HexToHash(poolID), common.Hash(args[0].([32]byte)))
			return pack(common.HexToAddress(fwWETH), common.HexToAddress(fwUSDT), common.HexToAddress(pair))
		case "token0":
			assert.Equal(t, common.HexToAddress(pair), target)
			return pack(common.HexToAddress(c.pairToken0))
		case "getReserves":
			assert.Equal(t, common.HexToAddress(pair), target)
			return pack(big10(c.reserve0), big10(c.reserve1), uint32(1))
		case "paused":
			assert.Equal(t, HookAddresses[0], target)
			return pack(c.paused)
		case "balanceOf":
			args, err := method.Inputs.Unpack(data[4:])
			require.NoError(t, err)
			holder := args[0].(common.Address)
			balance, ok := c.balances[target.Hex()][holder.Hex()]
			if !ok { // an address the test never funds holds nothing
				balance = "0"
			}
			return pack(big10(balance))
		}
		require.FailNow(t, "unexpected call", method.Name)
		return false, nil
	}
}

func lowerKeys(c *chain) *chain {
	out := map[string]map[string]string{}
	for token, holders := range c.balances {
		out[common.HexToAddress(token).Hex()] = map[string]string{}
		for holder, v := range holders {
			out[common.HexToAddress(token).Hex()][common.HexToAddress(holder).Hex()] = v
		}
	}
	c.balances = out
	return c
}

func track(t testing.TB, c *chain, h *Hook, block int64) (json.RawMessage, *testutil.MulticallStub, error) {
	t.Helper()
	stub := testutil.NewMulticallStub(t, lowerKeys(c).respond(t)).At(100, 1_700_000_000)
	p := newPool(t, nil, livePF)
	raw, err := h.Track(context.Background(), &uniswapv4.HookParam{
		Cfg: &uniswapv4.Config{ChainID: valueobject.ChainIDEthereum}, RpcClient: stub.Client(), Pool: &p,
		HookAddress: HookAddresses[0], BlockNumber: big.NewInt(block),
	})
	return raw, stub, err
}

func newHook() *Hook { return &Hook{} }

// The first track reads the route, then the state: two multicalls, the second pinned to the same block.
func TestTrack_FirstReadsRouteThenState(t *testing.T) {
	t.Parallel()
	h := newHook()
	raw, stub, err := track(t, liveChain(), h, 100)
	require.NoError(t, err)
	assert.Len(t, stub.Targets(), 2, "route, then state")

	var got Extra
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, liveExtra().Pair, got.Pair)
	assert.Equal(t, liveExtra().Few, got.Few)
	assert.Equal(t, [2]string{"16929650201529705921438", "29261029832819"}, [2]string{got.Reserves[0].Dec(), got.Reserves[1].Dec()})
	assert.Equal(t, [2]string{"564000000000000000000", "72000000000000"}, [2]string{got.Inventory[0].Dec(), got.Inventory[1].Dec()},
		"inventory is the PoolManager's balance of each currency")
	assert.Equal(t, [2]string{"1833000000000000000000", "1978000000000000"}, [2]string{got.Backing[0].Dec(), got.Backing[1].Dec()},
		"backing is each currency's balance at its Few token")
	assert.False(t, got.Paused)
	assert.Equal(t, got, h.Extra, "the hook keeps what it returned")

	// What Track stores is what a simulator quotes from.
	pl := newPool(t, json.RawMessage(raw), livePF)
	sim, err := uniswapv4.NewPoolSimulator(pl, valueobject.ChainIDEthereum)
	require.NoError(t, err)
	res, err := calcOut(t, sim, weth, usdt, "1000000000000")
	require.NoError(t, err)
	assert.Equal(t, "1721", res.TokenAmountOut.Amount.String())
}

// The route never changes, so a hook that has it skips the route read.
func TestTrack_KnownRouteSkipsTheRouteRead(t *testing.T) {
	t.Parallel()
	h := newHook()
	h.Extra = liveExtra()
	_, stub, err := track(t, liveChain(), h, 100)
	require.NoError(t, err)
	require.Len(t, stub.Targets(), 1)
	hookReads := 0
	for _, target := range stub.Targets()[0] {
		if target == HookAddresses[0] {
			hookReads++
		}
	}
	assert.Equal(t, 1, hookReads, "paused() is the only read of the hook itself, not registeredRouteFor")
	assert.Len(t, stub.Targets()[0], 7, "token0, getReserves, paused and four balances")
}

// The reads are pinned to the tracker's block: a read at another block fails.
func TestTrack_ReadsAtTheTrackersBlock(t *testing.T) {
	t.Parallel()
	_, _, err := track(t, liveChain(), newHook(), 99)
	assert.Error(t, err)
	_, _, err = track(t, liveChain(), newHook(), 100)
	assert.NoError(t, err)
}

// The pair may hold the Few tokens in either order; the reserves are stored in the pool's currency order. (Neither live
// pool has the pair's token0 as Few[1]: this is the only test of that branch.)
func TestTrack_ReversedPairOrder(t *testing.T) {
	t.Parallel()
	c := liveChain()
	c.pairToken0 = fwUSDT
	c.reserve0, c.reserve1 = "29261029832819", "16929650201529705921438" // the pair's token0 is fwUSDT
	raw, _, err := track(t, c, newHook(), 100)
	require.NoError(t, err)
	var got Extra
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "16929650201529705921438", got.Reserves[0].Dec())
	assert.Equal(t, "29261029832819", got.Reserves[1].Dec())
}

func TestTrack_PairOfOtherTokensIsRejected(t *testing.T) {
	t.Parallel()
	c := liveChain()
	c.pairToken0 = "0x2222222222222222222222222222222222222222"
	_, _, err := track(t, c, newHook(), 100)
	assert.ErrorIs(t, err, ErrPairMismatch)
}

func TestTrack_Paused(t *testing.T) {
	t.Parallel()
	c := liveChain()
	c.paused = true
	h := newHook()
	raw, _, err := track(t, c, h, 100)
	require.NoError(t, err)
	var got Extra
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.True(t, got.Paused)
	// A paused hook is tracked, and quotes refuse.
	sim, err := uniswapv4.NewPoolSimulator(newPool(t, json.RawMessage(raw), livePF), valueobject.ChainIDEthereum)
	require.NoError(t, err)
	_, err = calcOut(t, sim, weth, usdt, "1000000000000")
	assert.ErrorIs(t, err, ErrPaused)
}

func TestTrack_NoClient(t *testing.T) {
	t.Parallel()
	_, err := newHook().Track(context.Background(), &uniswapv4.HookParam{Pool: &entity.Pool{}})
	assert.ErrorIs(t, err, ErrPoolIsNotTracked)
}
