package dualpool

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Twofold NVDA/USDG on Robinhood Chain (chain 4663), hook 0xd1Bc…2Ac0, fee 500 /
// tickSpacing 10, snapshot at block 55199380 (2026-09-05). Expected outputs are
// the V4 Quoter's answers at that block:
//
//	quoteExactInputSingle(USDG->NVDA, 100 USDG)  = 421126678297991910
//	quoteExactInputSingle(NVDA->USDG, 0.1 NVDA)  = 22929103
//	quoteExactInputSingle(NVDA->USDG, 0.5 NVDA)  = 111778632
//	quoteExactInputSingle(USDG->NVDA, 1000 USDG) reverts (exceeds the deployed buckets)
var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0x085e812d2b072f9569c192769fef074eac9b3b519647665c98a7e4119f8fa06b","exchange":"uniswap-v4-dualpool","type":"uniswap-v4","timestamp":1788618000,"reserves":["263339524","1140173744005663979"],"tokens":[{"address":"0x5fc5360d0400a0fd4f2af552add042d716f1d168","symbol":"USDG","decimals":6,"swappable":true},{"address":"0xd0601ce157db5bdc3162bbac2a2c8af5320d9eec","symbol":"NVDA","decimals":18,"swappable":true}],"extra":"{\"liquidity\":0,\"sqrtPriceX96\":5214115250030972493292480690016563,\"tickSpacing\":10,\"tick\":221902,\"ticks\":[],\"hX\":{\"live\":true,\"b0\":\"0xfb23e04\",\"b1\":\"0xfd2b5fab1af4ceb\",\"sP\":\"0x1011363067154f9d3faa462565133\",\"t\":221902,\"pF\":0,\"lF\":500,\"bk\":[{\"l\":220700,\"u\":223100,\"w\":7000},{\"l\":217700,\"u\":226100,\"w\":3000}]}}","staticExtra":"{\"0x0\":[false,false],\"fee\":500,\"tS\":10,\"hooks\":\"0xd1bcbcca41f3bdb6b4812652959c6df725ea2ac0\",\"uR\":\"0x8876789976decbfcbbbe364623c63652db8c0904\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":55199380}`),
		&entityPool)
	poolSim = lo.Must(uniswapv4.NewPoolSimulator(entityPool, valueobject.ChainIDRobinhood))
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolSim, map[int]map[int]map[string]string{
		0: {1: {
			"100000000": "421126678297991910",
		}},
		1: {0: {
			"100000000000000000": "22929103",
			"500000000000000000": "111778632",
		}},
	})
}

func TestPoolSimulator_ExceedsBuckets(t *testing.T) {
	t.Parallel()
	_, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: entityPool.Tokens[0].Address, Amount: big.NewInt(1_000_000_000)},
		TokenOut:      entityPool.Tokens[1].Address,
	})
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
}

func TestPoolSimulator_UpdateBalanceThenQuoteAgain(t *testing.T) {
	t.Parallel()
	sim := poolSim.CloneState()
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: entityPool.Tokens[0].Address, Amount: big.NewInt(100_000_000)},
		TokenOut:      entityPool.Tokens[1].Address,
	})
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: entityPool.Tokens[0].Address, Amount: big.NewInt(100_000_000)},
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
	})
	again, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: entityPool.Tokens[0].Address, Amount: big.NewInt(100_000_000)},
		TokenOut:      entityPool.Tokens[1].Address,
	})
	require.NoError(t, err)
	require.Equal(t, -1, again.TokenAmountOut.Amount.Cmp(res.TokenAmountOut.Amount), "second buy must be worse")
	// the original simulator is untouched
	orig, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: entityPool.Tokens[0].Address, Amount: big.NewInt(100_000_000)},
		TokenOut:      entityPool.Tokens[1].Address,
	})
	require.NoError(t, err)
	require.Equal(t, "421126678297991910", orig.TokenAmountOut.Amount.String())
}
