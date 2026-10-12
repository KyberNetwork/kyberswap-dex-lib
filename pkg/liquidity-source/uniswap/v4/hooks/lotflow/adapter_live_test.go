package lotflow

import (
	"bytes"
	"context"
	"math"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Live adapter test against the LIVE V17 venues on chain 4663 (read-only: eth_call only). Env-gated
// like PR 1740's hook_live_test.go: LOTFLOW_RPC = an RPC URL (the public
// https://rpc.mainnet.chain.robinhood.com is enough). One block is pinned; the adapter tracks it
// (Track -> HookExtra -> factory, exactly as pool-service) and every size is quoted by the adapter and
// by the on-chain V4Quoter at that block. The adapter quotes prefunded (Kyber's executor); a sell the
// quoter refuses for float (it does not prefund) is re-quoted with the backed prefund override.
// Required: 0 over-quotes. Reported: max deviation.

var liveVenues = []struct {
	name, hook, lot, poolID string
}{
	{"LOT-V17 Review", "0xb672a5d825F82dEA8574711d1dA696FcC48d2AE0", "0xc9f3812b88FA6036f31b4165A8fbA75ED50E80c9",
		"0xe15c32e49aa49f4f26492cdcb158e8067f407b581a0619e007b78ffd1a0b2f3e"},
	{"bWALLETPIPEDOGNVDA", "0x447184450e4615c2ec2545540B53F213fB9c2aE0", "0x8fb11F269B01a4253a73eD5Ebd607bb80F32DbCD",
		"0x13b17d68eb63d3ef335cb009b72c222d2442b34e013bda4c59c1bfe722944a74"},
}

const liveQuoter = "0x8Dc178eFB8111BB0973Dd9d722ebeFF267c98F94"
const livePM = "0x8366a39cc670b4001a1121b8f6a443a643e40951"

func TestAdapterLive(t *testing.T) {
	url := os.Getenv("LOTFLOW_RPC")
	if url == "" {
		t.Skip("LOTFLOW_RPC not set (live chain-4663 adapter test)")
	}
	ctx := context.Background()
	rc, err := rpc.DialContext(ctx, url)
	require.NoError(t, err)
	var bn hexutil.Big
	require.NoError(t, rc.CallContext(ctx, &bn, "eth_blockNumber"))
	block := new(big.Int).Sub(bn.ToInt(), big.NewInt(2))
	parityBlock = hexutil.EncodeBig(block)
	defer func() { parityBlock = "latest" }()
	t.Logf("pinned block %s", block)
	client := ethrpc.New(url).SetMulticallContract(common.HexToAddress(liveMulticall))

	over := 0
	var maxDev float64
	for vi, v := range liveVenues {
		usdgIs0 := addrLess(parityUsdg, v.lot)
		t0, t1 := lotUsdg(v.lot, usdgIs0)
		pool := &entity.Pool{Address: v.poolID, Tokens: []*entity.PoolToken{{Address: strings.ToLower(t0)},
			{Address: strings.ToLower(t1)}}}
		param := &uniswapv4.HookParam{RpcClient: client, Pool: pool, HookAddress: common.HexToAddress(v.hook),
			BlockNumber: block, Cfg: &uniswapv4.Config{ChainID: valueobject.ChainIDRobinhood}}
		raw, err := (&Hook{}).Track(ctx, param)
		require.NoError(t, err, v.name)
		param.HookExtra = uniswapv4.HookExtra(raw)
		h := uniswapv4.HookFactories[common.HexToAddress(v.hook)](param).(*Hook)
		require.True(t, h.Tracked, v.name)
		lotSlot := lotBalanceSlot(t, ctx, rc, v.lot, livePM)
		t.Logf("%s: %d constituents, mq=%v, standing=%v, PM LOT float %s", v.name, len(h.Market.Constituents),
			h.Market.Mq != nil, h.Market.Venue.Standing != nil, h.Market.PmLotBalance)

		rng := rand.New(rand.NewSource(20261009 + int64(vi)))
		stats := map[string][4]int{} // side -> exact, below, bothRevert, adapterRefusedOnly
		for _, buy := range []bool{true, false} {
			side := map[bool]string{true: "buy", false: "sell"}[buy]
			st := stats[side]
			for k := 0; k < 30; k++ {
				x := math.Pow(10, rng.Float64()*math.Log10(2e4)) // $1..$20k / 1..20k LOT
				unit := big.NewFloat(1e6)
				if !buy {
					unit = big.NewFloat(1e18)
				}
				amt, _ := new(big.Float).Mul(big.NewFloat(x), unit).Int(nil)
				zfo := buy == usdgIs0
				r, aerr := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: zfo, AmountSpecified: amt})
				q, qrev, qerr := quoteV4(ctx, rc, liveQuoter, v.lot, v.hook, usdgIs0, zfo, amt, nil)
				if !buy && qerr != nil && bytes.Contains(qrev, sel("SellExceedsFloat(uint256,uint256)")) {
					cur, err := balanceOf(ctx, rc, v.lot, livePM)
					require.NoError(t, err)
					ov, err := backedFloatOverride(ctx, rc, v.lot, livePM, lotSlot, cur, amt, h.Market)
					require.NoError(t, err)
					q, _, qerr = quoteV4(ctx, rc, liveQuoter, v.lot, v.hook, usdgIs0, zfo, amt, ov)
				}
				switch {
				case aerr != nil && qerr != nil:
					st[2]++
				case aerr != nil:
					st[3]++ // refusing is never an over-quote
				case qerr != nil:
					over++
					t.Errorf("OVER %s %s %s: adapter fills %s, chain refuses", v.name, side, amt, r.DeltaUnspecified)
				default:
					out := new(big.Int).Neg(r.DeltaUnspecified)
					d, _ := new(big.Float).Quo(new(big.Float).SetInt(new(big.Int).Sub(out, q)), new(big.Float).SetInt(q)).Float64()
					maxDev = math.Max(maxDev, math.Abs(d*1e4))
					switch out.Cmp(q) {
					case 0:
						st[0]++
					case -1:
						st[1]++
					default:
						over++
						t.Errorf("OVER %s %s %s: adapter %s > quoter %s", v.name, side, amt, out, q)
					}
				}
			}
			stats[side] = st
		}
		t.Logf("%s: buy exact/below/bothRevert/adapterRefused = %v; sell = %v", v.name, stats["buy"], stats["sell"])
	}
	t.Logf("live adapter: over-quotes %d, max |deviation| %.4f bps", over, maxDev)
	require.Zero(t, over)
}
