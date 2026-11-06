package dexv2

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestLiveSnapshot(t *testing.T) {
	rpc := os.Getenv("FLUID_DEX_V2_RPC")
	if rpc == "" {
		t.Skip("set FLUID_DEX_V2_RPC for live test")
	}
	c := ethrpc.New(rpc)
	c.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))
	tracker := NewPoolTracker(&Config{Dex: "0xaBE4D5AA1Bb188f4B5d3a1aE3B8ec5e300A78261", Resolver: "0x143c3927EaF0e50a59b1651C2Eb1e37FB62Aa2a9", Liquidity: "0x52Aa899454998Be5b000Ad077a46Bbe360F4e497"}, c, nil)
	p := entity.Pool{Address: "0xe018f09af38956affdfeab72c2cefbcd4e6fee44d09df7525ec9dba3e51356a5_3", StaticExtra: `{"dex":"0xaBE4D5AA1Bb188f4B5d3a1aE3B8ec5e300A78261","dexType":3,"trackLiquidityLimits":true,"fee":100,"tickSpacing":1}`, Tokens: []*entity.PoolToken{{Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"}, {Address: "0xdac17f958d2ee523a2206206994597c13d831ec7"}}}
	x, err := tracker.fetchRPCData(context.Background(), p, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(x.SnapshotBlock, x.SnapshotTimestamp)
	t.Log("float", x.FloatBalance, "cap", x.MaxCap, "unaccounted", x.UnaccountedBorrow, "withdraw", x.SwapOutWithdraw, "borrow", x.SwapOutBorrow)
	ticks, err := tracker.fetchPoolTicksFromBitmap(context.Background(), p, x.SnapshotBlock)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ticks", len(ticks))
	x.Ticks = ticks
	t.Log("snapshot", x.SnapshotBlock, "current tick", x.Tick, "ticks", len(ticks))
	data, _ := json.Marshal(x)
	t.Log("state", x.Tick, x.SqrtPriceX96, "ticks", ticks)
	p.Extra = string(data)
	p.Reserves = []string{"100000000000", "100000000000"}
	p.BlockNumber = x.SnapshotBlock
	p.Tokens[0].Decimals = 6
	p.Tokens[1].Decimals = 6
	sim, e := NewPoolSimulator(p, valueobject.ChainIDEthereum)
	if e != nil {
		t.Fatal(e)
	}
	for _, i := range []int{0, 1} {
		amount := int64(1000000000)
		if i == 1 {
			amount = 10000000
		}
		q, e := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[i].Address, Amount: big.NewInt(amount)}, TokenOut: p.Tokens[1-i].Address})
		if e != nil {
			t.Log("quote", i, e)
		} else {
			t.Log("quote", i, q.TokenAmountOut.Amount)
		}
	}
}

// TestLegacyPolygonSnapshot checks the resolver deployed before token-base state
// was added. It intentionally exercises the legacy untracked settlement path.
func TestLegacyPolygonSnapshot(t *testing.T) {
	rpc := os.Getenv("FLUID_DEX_V2_POLYGON_RPC")
	if rpc == "" {
		t.Skip("set FLUID_DEX_V2_POLYGON_RPC for live test")
	}
	c := ethrpc.New(rpc)
	c.SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	tracker := NewPoolTracker(&Config{Dex: "0x7822B3944B1a68B231a6e7F55B57967F28BB369e", Resolver: "0x1E45589D501AcED82013c2838552122f943B33Ac", Liquidity: "0x52Aa899454998Be5b000Ad077a46Bbe360F4e497"}, c, nil)
	p := entity.Pool{Address: "0x8d1b5f8da63fa29b191672231d3845740a11fcbef6c76e077cfffe56cc27c707_3", StaticExtra: `{"dex":"0x7822B3944B1a68B231a6e7F55B57967F28BB369e","dexType":3,"fee":100,"tickSpacing":1}`, Tokens: []*entity.PoolToken{{Address: "0x3c499c542cef5e3811e1192ce70d8cc03d5c3359", Decimals: 6}, {Address: "0xc2132d05d31c914a87c6611c10748aeb04b58e8f", Decimals: 6}}}
	x, err := tracker.fetchRPCData(context.Background(), p, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	ticks, err := tracker.fetchPoolTicksFromBitmap(context.Background(), p, x.SnapshotBlock)
	if err != nil {
		t.Fatal(err)
	}
	x.Ticks = ticks
	t.Log("snapshot", x.SnapshotBlock, "current tick", x.Tick, "ticks", len(ticks))
	data, _ := json.Marshal(x)
	p.Extra = string(data)
	p.BlockNumber = x.SnapshotBlock
	p.Reserves = []string{"100000000000", "100000000000"}
	sim, err := NewPoolSimulator(p, valueobject.ChainIDPolygon)
	if err != nil {
		t.Fatal(err)
	}
	q, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[0].Address, Amount: big.NewInt(1000000)}, TokenOut: p.Tokens[1].Address})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("block", x.SnapshotBlock, "ticks", len(ticks), "output", q.TokenAmountOut.Amount)
}
