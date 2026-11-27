package fewshell

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	// The WETH/USDT shell pool on Ethereum, shaped as the router API serves it (a full-range decoy position).
	poolID = "0xff2441552bc87bf82d0647f28f51e8a218218ff6a0fd7bee865359276fea3ef2"
	weth   = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	usdt   = "0xdac17f958d2ee523a2206206994597c13d831ec7"

	fwWETH = "0xa250cc729bb3323e7933022a67b52200fe354767"
	fwUSDT = "0xef87f4608e601e8564800265aee1c1ffadf73283"
	pair   = "0x147d15e009a63ebed5196ea029679329204f98fd"

	// pF of the live pool: lpFee 3000, both directions' protocol fee 40 (40*25 = 1000 ppm, 10 bps).
	livePF uint64 = 3000<<24 | 40<<12 | 40
	// calibratorHX is the private calibrator's hX the live pool carried before this hook was registered.
	calibratorHX = `{"f":[{"s":[{"x":26.759555296156933,"o":0.2772772772772773},{"x":38.281260096891295}]},{}],"b":1536,` +
		`"lp":[277279,1],"cl":[105955037,0],"u":1795754346,"k":8,"v":1795755334,` +
		`"a":[[{"i":"418337891649","r":"277277"}],[{"i":"67103","r":"0"}]],"r":[3109,1277]}`
)

func u(s string) uint256.Int {
	v, err := uint256.FromDecimal(s)
	if err != nil {
		panic(err)
	}
	return *v
}

// liveExtra is the tracked state at block 26504066: the pair reserves the brief's on-chain outputs come from, and
// inventories and backings far above any amount the tests swap.
func liveExtra() Extra {
	return Extra{
		Pair:      common.HexToAddress(pair),
		Few:       [2]common.Address{common.HexToAddress(fwWETH), common.HexToAddress(fwUSDT)},
		Reserves:  [2]uint256.Int{u("16929650201529705921438"), u("29261029832819")},
		Inventory: [2]uint256.Int{u("564000000000000000000"), u("72000000000000")},
		Backing:   [2]uint256.Int{u("1833000000000000000000"), u("1978000000000000")},
	}
}

// newPool builds the pool entity with hX = hx (any value; nil for none) and pF = pF (0 for none).
func newPool(t testing.TB, hx any, pF uint64) entity.Pool {
	t.Helper()
	return newPoolWithTicks(t, hx, pF, -887272, 887272)
}

// newPoolWithTicks is newPool with the decoy position between two ticks instead of full range.
func newPoolWithTicks(t testing.TB, hx any, pF uint64, lower, upper int) entity.Pool {
	t.Helper()
	extra := map[string]any{
		"liquidity": 20422207567, "sqrtPriceX96": json.RawMessage("3879510197480991806181244"),
		"tickSpacing": 1, "tick": -198498,
		"ticks": json.RawMessage(fmt.Sprintf(`[{"index":%d,"liquidityGross":20422207567,"liquidityNet":20422207567},`+
			`{"index":%d,"liquidityGross":20422207567,"liquidityNet":-20422207567}]`, lower, upper)),
	}
	if pF != 0 {
		extra["pF"] = pF
	}
	switch v := hx.(type) {
	case nil:
	case string:
		extra["hX"] = json.RawMessage(v)
	default:
		raw, err := json.Marshal(v)
		require.NoError(t, err)
		extra["hX"] = json.RawMessage(raw)
	}
	rawExtra, err := json.Marshal(extra)
	require.NoError(t, err)
	return entity.Pool{
		Address: poolID, Exchange: "uniswap-v4-fee", Type: "uniswap-v4", SwapFee: 3040,
		Reserves: pseudoTVL(),
		Tokens: []*entity.PoolToken{
			{Address: weth, Symbol: "WETH", Decimals: 18, Swappable: true},
			{Address: usdt, Symbol: "USDT", Decimals: 6, Swappable: true},
		},
		Extra: string(rawExtra),
		StaticExtra: `{"0x0":[false,false],"fee":3000,"tS":1,"hooks":"0xb39aa0fea3d8940c1106059460934ccfb0212088",` +
			`"uR":"0x66a9893cc07d91d95644aedd05d03f95e1dba8af","pm2":"0x000000000022d473030f116ddee9f6b43ac78ba3",` +
			`"mc3":"0xca11bde05977b3631167028862be2a173976ca11"}`,
	}
}

func newSim(t testing.TB, hx any, pF uint64) *uniswapv4.PoolSimulator {
	t.Helper()
	sim, err := uniswapv4.NewPoolSimulator(newPool(t, hx, pF), valueobject.ChainIDEthereum)
	require.NoError(t, err)
	return sim
}

func big10(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic(s)
	}
	return v
}

// pseudoTVL is what the tracker stores as the pool's reserves for liveExtra: the hook's GetReserves, not the decoy
// position's 417066561925500 / 999999.
func pseudoTVL() []string {
	reserves, err := (&Hook{Extra: liveExtra()}).GetReserves(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	return reserves
}
