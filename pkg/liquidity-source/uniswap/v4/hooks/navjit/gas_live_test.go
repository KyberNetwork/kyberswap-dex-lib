package navjit

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// measureCap is far above any rung, so a rung that fills reports its true gas and one that cannot
// fill burns this much (one call per rung keeps the eth_call under the node's 50M cap).
const measureCap = 20_000_000

// TestLive_QuoteGas logs the gas of every ladder rung on each live venue plus LOT-oldy (the thin
// one), one sub-call per eth_call. quoteGasCap must sit above the largest filling gas it prints.
func TestLive_QuoteGas(t *testing.T) {
	skipInCI(t)
	ctx := context.Background()
	rpc := liveRPC()
	quoter := QuoterByChain[valueobject.ChainIDRobinhood]
	specs := append([]poolSpec{{Name: "LOT-oldy",
		ID:        "0x85d90dac41ad4205b80b4c4e14f4831805fc7fe3e84068a59c7a2618e4db1817",
		Currency0: usdgAddr, Currency1: "0x8bb6f07aba0dae22943adc96338cd297000442c2",
		Dec0: usdgDecimals, Dec1: 18, Fee: 0, TickSpacing: 1, Hooks: HookAddresses[0].Hex()}}, liveVenues...)

	var maxFill uint64
	for _, v := range specs {
		key := quoteParams{HookData: []byte{}}
		key.PoolKey.Currency0, key.PoolKey.Currency1 = common.HexToAddress(v.Currency0), common.HexToAddress(v.Currency1)
		key.PoolKey.Hooks = HookAddresses[0]
		key.PoolKey.Fee, key.PoolKey.TickSpacing = big.NewInt(int64(v.Fee)), big.NewInt(int64(v.TickSpacing))

		// nav and float via the same multicall, to size the sells exactly as Track does.
		hook, lot := HookAddresses[0], common.HexToAddress(v.Currency1)
		var navGuard, pm common.Address
		_, err := rpc.NewRequest().SetContext(ctx).
			AddCall(&ethrpc.Call{ABI: hookABI, Target: hexutil.Encode(hook[:]), Method: "navGuard"}, []any{&navGuard}).
			AddCall(&ethrpc.Call{ABI: hookABI, Target: hexutil.Encode(hook[:]), Method: "poolManager"}, []any{&pm}).Aggregate()
		require.NoError(t, err)
		nc, err := newCappedCall(navGuardABI, navGuard, "checkedNavPerUnit18", lot)
		require.NoError(t, err)
		fc, err := newCappedCall(erc20ABI, lot, "balanceOf", pm)
		require.NoError(t, err)
		nc.GasLimit, fc.GasLimit = big.NewInt(measureCap), big.NewInt(measureCap)
		res, err := cappedMulticall(ctx, rpc, nil, nil, []cappedCall{nc, fc})
		require.NoError(t, err)
		var nav, float *big.Int
		require.True(t, res[0].unpack(navGuardABI, "checkedNavPerUnit18", &nav))
		require.True(t, res[1].unpack(erc20ABI, "balanceOf", &float))
		t.Logf("%s nav %v float %v nav-call gas %v float-call gas %v", v.Name, nav, float, res[0].GasUsed, res[1].GasUsed)

		buyIn := make([]*big.Int, len(rungsUsd))
		var sellIn []*big.Int
		for i, usd := range rungsUsd {
			buyIn[i] = new(big.Int).Mul(new(big.Int).SetUint64(usd), bignumber.TenPowInt(int(v.Dec0)))
			amt := new(big.Int).Mul(new(big.Int).SetUint64(usd), bignumber.TenPowInt(36))
			if amt.Div(amt, nav).Cmp(float) < 0 {
				sellIn = append(sellIn, amt)
			}
		}
		for _, fb := range floatRungsBps {
			sellIn = append(sellIn, new(big.Int).Div(new(big.Int).Mul(float, big.NewInt(fb)), bignumber.BasisPoint))
		}
		for _, d := range []struct {
			name string
			zfo  bool
			in   []*big.Int
		}{{"buy", true, buyIn}, {"sell", false, sellIn}} {
			for _, amt := range d.in {
				calls, err := appendLadder(nil, quoter, key, d.zfo, []*big.Int{amt})
				require.NoError(t, err)
				calls[0].GasLimit = big.NewInt(measureCap)
				res, err := cappedMulticall(ctx, rpc, nil, nil, calls)
				require.NoError(t, err)
				t.Logf("%s %s in=%v ok=%v gasUsed=%v", v.Name, d.name, amt, res[0].Success, res[0].GasUsed)
				if res[0].Success {
					maxFill = max(maxFill, res[0].GasUsed.Uint64())
				}
			}
		}
	}
	t.Logf("max filling gasUsed = %d", maxFill)
}

// TestLive_ThinVenueTracks: LOT-oldy has 9 failing buy rungs, which used to run the one-call ladder
// out of gas (pool never tracked). With the per-sub-call cap one eth_call per round tracks it.
func TestLive_ThinVenueTracks(t *testing.T) {
	skipInCI(t)
	v := poolSpec{Name: "LOT-oldy", ID: "0x85d90dac41ad4205b80b4c4e14f4831805fc7fe3e84068a59c7a2618e4db1817",
		Currency0: usdgAddr, Currency1: "0x8bb6f07aba0dae22943adc96338cd297000442c2",
		Dec0: usdgDecimals, Dec1: 18, Fee: 0, TickSpacing: 1, Hooks: HookAddresses[0].Hex()}
	p := trackPool(context.Background(), t, liveRPC(), v)
	var ex uniswapv4.Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &ex))
	var hx Extra
	require.NoError(t, uniswapv4.HookExtra(ex.HookExtra).Unmarshal(&hx))
	t.Logf("tracked=%v block=%d buy=%d sell=%d", hx.Tracked, p.BlockNumber, len(hx.Buy), len(hx.Sell))
	require.True(t, hx.Tracked)
	require.NotEmpty(t, hx.Buy)
	require.NotEmpty(t, hx.Sell)
}
