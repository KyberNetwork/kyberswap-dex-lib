package apefun

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	v3utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The local simulator must reproduce, to the unit, what the chain paid on real ape.fun swaps routed straight to
// the PoolManager (a KyberSwap executor and other routers: sender is not the ape.fun router, so recipient is 0 and
// never snipe-exempt), and Uniswap's V4Quoter (0x8dc178efb8111bb0973dd9d722ebeff267c98f94 on both chains, which
// runs the real hook through PoolManager.unlock). Pool state before each swap was rebuilt from the pool's
// Initialize, ModifyLiquidity and Swap events and checked against PoolManager.extsload at the previous block; the
// hook extra is what Track read live (policies, buybackVaults, launchForToken). Every pool has the quote asset as
// currency0 and one position from MIN_TICK to its opening tick, as the factory creates it.
// Pure fixtures, no network: these run in CI.

type launch struct {
	chainID     valueobject.ChainID
	hook        common.Address
	pool, quote string
	coin        string
	extra       Extra
}

var (
	usdcArc = "0x3600000000000000000000000000000000000000"
	wethRh  = "0x0bd7d308f8e1639fab988df18a8011f41eacad73"

	arc1br = launch{valueobject.ChainIDArc, HookArc, "0x75e34e5455e5a8baf744d93444efef26c85c051f4aa51b8b41d12e130902c178",
		usdcArc, "0x517a96dd781bd45d8a9841907cb045acfca0e7ae",
		Extra{Live: true, QuoteIsCurrency0: true, TaxBps: 100, Launched: 1791413898, BuybackVault: true}}
	arc2reward = launch{valueobject.ChainIDArc, HookArc, "0x9aad681373e1ef45fe09c32db8acd2f395472e8c414bcf8f13d0adc7469911c4",
		usdcArc, "0xf2c8da573568851cac5a129f3ea54d9ba8b11f9a",
		Extra{Live: true, QuoteIsCurrency0: true, TaxBps: 100, Launched: 1791414828}}
	arc3cr = launch{valueobject.ChainIDArc, HookArc, "0x03e0f5ceda487e800cff557d7901cd027f66da50f73bb77b1ce4c247dee2c23c",
		usdcArc, "0x66aae54fa60084ef29ded860154698c268606872",
		Extra{Live: true, QuoteIsCurrency0: true, Launched: 1791415158}}
	rh1 = launch{valueobject.ChainIDRobinhood, HookRobinhood, "0x6c0938544d2ac7e67a8e87364de170b2d06197311fcef3a16895cc98e661e3b3",
		wethRh, "0x78ed0717004ff3c1457254ee34c78e82b58c54c7",
		Extra{Live: true, QuoteIsCurrency0: true, TaxBps: 100, Launched: 1791420973, BuybackVault: true}}
	rh4 = launch{valueobject.ChainIDRobinhood, HookRobinhood, "0x331dfe75a9151db444783fef00f2a0a7b97bc5682211f2283f3e13db6626d813",
		wethRh, "0x85a7c6615fc55d33c327641d9e87360c07a2b16f",
		Extra{Live: true, QuoteIsCurrency0: true, Launched: 1791454166}}
)

// poolState is the pool's slot0 and liquidity, and the upper edge of its one position (the opening tick).
type poolState struct {
	liquidity, sqrtPriceX96 string
	tick, openingTick       int
}

func (l launch) sim(t testing.TB, s poolState, x Extra, hooks common.Address) *uniswapv4.PoolSimulator {
	t.Helper()
	hx, err := json.Marshal(x)
	require.NoError(t, err)
	pool := entity.Pool{
		Address:  l.pool,
		Exchange: string(valueobject.ExchangeUniswapV4ApeFun),
		Type:     uniswapv4.DexType,
		Tokens:   []*entity.PoolToken{{Address: l.quote, Swappable: true}, {Address: l.coin, Swappable: true}},
		// reserves only cap swap sizes in the simulator; the pool's real balances sit in the PoolManager
		Reserves: entity.PoolReserves{"1000000000000000000000000000000000", "1000000000000000000000000000000000"},
		StaticExtra: `{"0x0":[false,false],"fee":0,"tS":1,"hooks":"` + hexutil.Encode(hooks[:]) +
			`","uR":"0x0000000000000000000000000000000000000000","pm2":"0x0000000000000000000000000000000000000000",` +
			`"mc3":"0xcA11bde05977b3631167028862bE2a173976CA11"}`,
		Extra: fmt.Sprintf(`{"liquidity":%[1]s,"sqrtPriceX96":%[2]s,"tickSpacing":1,"tick":%[3]d,"ticks":[`+
			`{"index":-887272,"liquidityGross":%[1]s,"liquidityNet":%[1]s},`+
			`{"index":%[4]d,"liquidityGross":%[1]s,"liquidityNet":-%[1]s}],"hX":%[5]s}`,
			s.liquidity, s.sqrtPriceX96, s.tick, s.openingTick, hx),
	}
	if hooks == (common.Address{}) {
		pool.Exchange = "uniswap-v4"
	}
	sim, err := uniswapv4.NewPoolSimulator(pool, l.chainID)
	require.NoError(t, err)
	return sim
}

func (l launch) tokens(buy bool) (in, out string) {
	if buy {
		return l.quote, l.coin
	}
	return l.coin, l.quote
}

func quoteOut(t testing.TB, sim *uniswapv4.PoolSimulator, tokenIn, tokenOut string, amount *big.Int) (*big.Int, error) {
	t.Helper()
	res, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{Token: tokenIn, Amount: new(big.Int).Set(amount)}, TokenOut: tokenOut})
	if err != nil {
		return nil, err
	}
	return res.TokenAmountOut.Amount, nil
}

// Real swaps on both chains, the snipe window included: the buys at 0 s and 1 s paid 99% and 6.18% on top of 1%.
// want is what the trader received (FeeAccrued.grossQuote less fee and tax on sells).
func TestDirectProtocolMatch_RealSwaps(t *testing.T) {
	for _, c := range []struct {
		name, tx string
		l        launch
		time     int64
		buy      bool
		in, want string
		state    poolState
	}{
		{"ARC1BR buy at 0 s", "0xbb47a7523c398400a36e884975486c9c53cbc9ebec343121a9c3e388c4efaa05", arc1br, 1791413898, true, "5000", "15404375965258094718",
			poolState{"1798895067137432842", "43976092731357914839644534793867190970", 402711, 402742}},
		{"ARC2REWARD buy at 1 s", "0xb97a505432c7c2cb34c2b2573766b2d40a8959c68f21476e6cba2644be364e51", arc2reward, 1791414829, true, "5000", "1414427819313178888439",
			poolState{"1798895067137432842", "43976092731357914839644534793867190970", 402711, 402742}},
		{"ARC3CR buy at 1 s", "0xdb4109650cfd23b6cc5a4b408c4b7dc39a748b73402772841d7ce8fe21622a8a", arc3cr, 1791415159, true, "5000", "1429788034640785900484",
			poolState{"1798895067137432842", "43975414291789692221835794799516761535", 402711, 402742}},
		{"ARC3CR sell at 849 s", "0x39b5c4d313687cd436bce0d6c55d5eb9c4230713cceffd8ae47d57eb3ee0a2e8", arc3cr, 1791416007, false, "150950376041770553067228", "485149",
			poolState{"1798895067137432842", "43968703060754628274701709567747613033", 402708, 402742}},
		{"ARC2REWARD sell at 1259 s", "0x428c9e7c48cd8ee514aa882403f655da56d4975d1cb96806e4bfa05fd2a721a5", arc2reward, 1791416087, false, "1045074858624869621428512", "3327787",
			poolState{"1798895067137432842", "43930002537010018541724718686399501459", 402690, 402742}},
		{"RH4 buy at 84 s", "0x75866dbcd4f2ad853d58d096ecc8deded3761479f725d16b648869550ef7381c", rh4, 1791454250, true, "495000000000000", "276238798519789493104850",
			poolState{"40987439322977844309085", "1881323003042868532779145202567584", 201513, 202055}},
		{"RH4 sell at 91 s", "0xb04af3ae7c33685e7fa0e8f6082569667aa92967e3c167b995e24c13de594287", rh4, 1791454257, false, "276238798519789493104850", "485149500000000",
			poolState{"40987439322977844309085", "1880789037203982822712555627092378", 201507, 202055}},
		{"RH1 buy at 49870 s", "0x67c7247047d9729df3546d29af2694796cd921dd455da68c61745dbfda40c01d", rh1, 1791470843, true, "7900236222200242", "4577016687188976589220060",
			poolState{"40987439322977844309085", "1930790080765937457894462303657963", 202032, 202055}},
		{"RH1 sell at 49912 s", "0xa444f0bfd1d007a3ab9cf9dc86d437f1b16bcb07dc8af388bcd5e952c0cc0dc0", rh1, 1791470885, false, "4577016687188976589220060", "7587386867801113",
			poolState{"40987439322977844309085", "1921942769786941610198572078238218", 201940, 202055}},
	} {
		t.Run(c.name, func(t *testing.T) {
			// the quoting clock is read clockMarginSeconds early: a clock that far ahead of the block is exact
			pinClock(t, c.time+clockMarginSeconds)
			tokenIn, tokenOut := c.l.tokens(c.buy)
			got, err := quoteOut(t, c.l.sim(t, c.state, c.l.extra, c.l.hook), tokenIn, tokenOut, bi(t, c.in))
			require.NoError(t, err)
			assert.Equal(t, c.want, got.String(), c.tx)
		})
	}
}

// With the clock equal to the block timestamp (no margin used), the three real buys inside the window are quoted
// at or below what the trader received: the quote is conservative, never generous.
func TestSnipeWindow_ConservativeAtBlockTime(t *testing.T) {
	for _, c := range []struct {
		l      launch
		time   int64
		in     string
		actual string
		state  poolState
	}{
		{arc1br, 1791413898, "5000", "15404375965258094718", poolState{"1798895067137432842", "43976092731357914839644534793867190970", 402711, 402742}},
		{arc2reward, 1791414829, "5000", "1414427819313178888439", poolState{"1798895067137432842", "43976092731357914839644534793867190970", 402711, 402742}},
		{arc3cr, 1791415159, "5000", "1429788034640785900484", poolState{"1798895067137432842", "43975414291789692221835794799516761535", 402711, 402742}},
	} {
		pinClock(t, c.time)
		got, err := quoteOut(t, c.l.sim(t, c.state, c.l.extra, c.l.hook), c.l.quote, c.l.coin, bi(t, c.in))
		require.NoError(t, err)
		assert.LessOrEqualf(t, got.Cmp(bi(t, c.actual)), 0, "%s: quoted %s, chain paid %s", c.l.coin, got, c.actual)
	}
}

// mirror builds the same RH4 pool with the coin as currency0 (every live pool has the quote asset as currency0
// since the quote addresses sort low; a coin address below the quote's would flip it): price inverted, the
// position from -opening to MAX_TICK, hook extra with QuoteIsCurrency0 false.
func mirror(t testing.TB, s poolState, x Extra, hooks common.Address) *uniswapv4.PoolSimulator {
	t.Helper()
	sqrtP := bi(t, s.sqrtPriceX96)
	inv := new(big.Int).Quo(new(big.Int).Lsh(bignumber.One, 192), sqrtP)
	tick, err := v3utils.GetTickAtSqrtRatio(inv)
	require.NoError(t, err)
	x.QuoteIsCurrency0 = false
	hx, err := json.Marshal(x)
	require.NoError(t, err)
	pool := entity.Pool{
		Address:  rh4.pool,
		Exchange: string(valueobject.ExchangeUniswapV4ApeFun),
		Type:     uniswapv4.DexType,
		Tokens:   []*entity.PoolToken{{Address: rh4.coin, Swappable: true}, {Address: rh4.quote, Swappable: true}},
		Reserves: entity.PoolReserves{"1000000000000000000000000000000000", "1000000000000000000000000000000000"},
		StaticExtra: `{"0x0":[false,false],"fee":0,"tS":1,"hooks":"` + hexutil.Encode(hooks[:]) +
			`","uR":"0x0000000000000000000000000000000000000000","pm2":"0x0000000000000000000000000000000000000000",` +
			`"mc3":"0xcA11bde05977b3631167028862bE2a173976CA11"}`,
		Extra: fmt.Sprintf(`{"liquidity":%[1]s,"sqrtPriceX96":%[2]s,"tickSpacing":1,"tick":%[3]d,"ticks":[`+
			`{"index":%[4]d,"liquidityGross":%[1]s,"liquidityNet":%[1]s},`+
			`{"index":887272,"liquidityGross":%[1]s,"liquidityNet":-%[1]s}],"hX":%[5]s}`,
			s.liquidity, inv, tick, -s.openingTick, hx),
	}
	sim, err := uniswapv4.NewPoolSimulator(pool, valueobject.ChainIDRobinhood)
	require.NoError(t, err)
	return sim
}

// Coin as currency0: buys are oneForZero and sells zeroForOne. The hooked pool still equals the hookless pool
// fed the net amounts (exact, every tax and window second), and agrees with the real orientation to 1e-12.
func TestMirroredOrientation_CoinIsCurrency0(t *testing.T) {
	state := poolState{"40987439322977844309085", "1881323003042868532779145197824480", 201513, 202055}
	for _, tax := range taxes {
		x := rh4.extra
		x.TaxBps = tax
		hooked, plain := mirror(t, state, x, rh4.hook), mirror(t, state, Extra{}, common.Address{})
		orig := rh4.sim(t, state, x, rh4.hook)
		for elapsed := int64(0); elapsed <= 4; elapsed++ {
			pinClock(t, rh4.extra.Launched+elapsed+clockMarginSeconds)
			for _, a := range []string{"1000000000000", "495000000000000", "7900236222200242"} {
				gross := bi(t, a)
				fee, taxAmt := solidityFees(gross, tax, elapsed, true)
				net := new(big.Int).Sub(gross, fee)
				net.Sub(net, taxAmt)
				want, err := quoteOut(t, plain, wethRh, rh4.coin, net)
				require.NoError(t, err)
				got, err := quoteOut(t, hooked, wethRh, rh4.coin, gross)
				require.NoError(t, err)
				assert.Equal(t, want.String(), got.String(), "buy tax %d elapsed %d gross %s", tax, elapsed, a)
				ref, err := quoteOut(t, orig, wethRh, rh4.coin, gross)
				require.NoError(t, err)
				assertClose(t, ref, got)
			}
			for _, a := range []string{"276238798519789493104850", "4577016687188976589220060"} {
				q, err := quoteOut(t, plain, rh4.coin, wethRh, bi(t, a))
				require.NoError(t, err)
				fee, taxAmt := solidityFees(q, tax, elapsed, false)
				want := new(big.Int).Sub(q, fee)
				want.Sub(want, taxAmt)
				got, err := quoteOut(t, hooked, rh4.coin, wethRh, bi(t, a))
				require.NoError(t, err)
				assert.Equal(t, want.String(), got.String(), "sell tax %d elapsed %d amount %s", tax, elapsed, a)
				ref, err := quoteOut(t, orig, rh4.coin, wethRh, bi(t, a))
				require.NoError(t, err)
				assertClose(t, ref, got)
			}
		}
	}
}

// assertClose: within 1e-12 relative (the mirrored price is the inverse rounded once).
func assertClose(t *testing.T, want, got *big.Int) {
	t.Helper()
	diff := new(big.Int).Abs(new(big.Int).Sub(want, got))
	assert.LessOrEqualf(t, new(big.Int).Mul(diff, bignumber.TenPowInt(12)).Cmp(want), 0, "want %s got %s", want, got)
}

// V4Quoter.quoteExactInputSingle against the simulator at one block per chain (Arc 25246007, timestamp 1791637722;
// Robinhood 85021170, timestamp 1791637725), state read by the uniswap-v4 PoolTracker itself (StateView slot0,
// liquidity and getTickInfo, then Track). A sell past the position's quote reverts in the quoter and errors here.
// Buys of 1 ETH and more cross many tick-bitmap words, where v4-core steps word by word and the shared v3 math
// does not; those differ by about 1e-14 (simulator lower) and are left out.
func TestDirectProtocolMatch_V4Quoter(t *testing.T) {
	for _, c := range []struct {
		l     launch
		time  int64
		state poolState
		cases [][3]string // buy|sell, amount in, quoter amountOut
	}{
		{arc2reward, 1791637722, poolState{"1798895067137432842", "43903652983428404847827045845214314515", 402678, 402742},
			[][3]string{
				{"buy", "10000", "3009309881353294267977"},
				{"buy", "1000000", "300841077237029842754191"},
				{"buy", "10000000", "3000261630691947972809879"},
				{"sell", "100000000000000000000", "319"},
				{"sell", "1000000000000000000000000", "3188222"},
			}},
		{rh4, 1791637725, poolState{"40987439322977844309085", "1881323003042868532779145197824480", 201513, 202055},
			[][3]string{
				{"buy", "10000000000000000", "5550332439725505417402886"},
				{"sell", "100000000000000000000", "175576984616"},
				{"sell", "1000000000000000000000000", "1753967892601352"},
			}},
	} {
		pinClock(t, c.time)
		sim := c.l.sim(t, c.state, c.l.extra, c.l.hook)
		for _, q := range c.cases {
			tokenIn, tokenOut := c.l.tokens(q[0] == "buy")
			got, err := quoteOut(t, sim, tokenIn, tokenOut, bi(t, q[1]))
			require.NoError(t, err)
			assert.Equal(t, q[2], got.String(), "%s %s %s", c.l.coin, q[0], q[1])
		}
		_, err := quoteOut(t, sim, c.l.coin, c.l.quote, bi(t, "100000000000000000000000000"))
		assert.Error(t, err, "1e8 coins is more than the position's quote")
	}
}

// The hook adds only its fees around a fee-0 pool: a buy equals the same hookless pool fed the gross input less
// fee and tax, a sell equals the hookless output less fee and tax on it, for tax 0, 1% and 10% and every second
// of the snipe window (fees from the independent Solidity port).
func TestFeesAroundHooklessPool(t *testing.T) {
	state := poolState{"40987439322977844309085", "1881323003042868532779145197824480", 201513, 202055}
	for _, tax := range taxes {
		x := rh4.extra
		x.TaxBps = tax
		hooked, plain := rh4.sim(t, state, x, rh4.hook), rh4.sim(t, state, Extra{}, common.Address{})
		for elapsed := int64(-1); elapsed <= 4; elapsed++ {
			pinClock(t, rh4.extra.Launched+elapsed+clockMarginSeconds)
			for _, a := range []string{"10000", "1000000000000", "495000000000000", "7900236222200242", "50000000000000000"} {
				gross := bi(t, a)
				fee, taxAmt := solidityFees(gross, tax, max(elapsed, 0), true)
				net := new(big.Int).Sub(gross, fee)
				net.Sub(net, taxAmt)
				want, err := quoteOut(t, plain, wethRh, rh4.coin, net)
				require.NoError(t, err)
				got, err := quoteOut(t, hooked, wethRh, rh4.coin, gross)
				require.NoError(t, err)
				assert.Equal(t, want.String(), got.String(), "buy tax %d elapsed %d gross %s", tax, elapsed, a)
			}
			for _, a := range []string{"1000000000000000000", "276238798519789493104850", "4577016687188976589220060"} {
				q, err := quoteOut(t, plain, rh4.coin, wethRh, bi(t, a))
				require.NoError(t, err)
				fee, taxAmt := solidityFees(q, tax, max(elapsed, 0), false)
				want := new(big.Int).Sub(q, fee)
				want.Sub(want, taxAmt)
				got, err := quoteOut(t, hooked, rh4.coin, wethRh, bi(t, a))
				require.NoError(t, err)
				assert.Equal(t, want.String(), got.String(), "sell tax %d elapsed %d amount %s", tax, elapsed, a)
			}
		}
	}
}

// CalcAmountIn reverses exact-in (the executor never swaps exact-out, which the hook refuses): the input it
// returns, quoted exact-in, delivers at least the requested output, in the snipe window and after.
func TestCalcAmountIn_ReversesExactIn(t *testing.T) {
	state := poolState{"1798895067137432842", "43903652983428404847827045845214314515", 402678, 402742}
	for _, tax := range taxes {
		x := arc2reward.extra
		x.TaxBps = tax
		sim := arc2reward.sim(t, state, x, arc2reward.hook)
		for _, elapsed := range []int64{0, 1, 2, 3, 600} {
			pinClock(t, x.Launched+elapsed)
			for _, c := range []struct {
				buy bool
				out string
			}{{true, "1000000000000000000"}, {true, "300841077237029842754191"}, {false, "319"}, {false, "3188222"}} {
				tokenIn, tokenOut := arc2reward.tokens(c.buy)
				res, err := sim.CalcAmountIn(poolpkg.CalcAmountInParams{
					TokenAmountOut: poolpkg.TokenAmount{Token: tokenOut, Amount: bi(t, c.out)}, TokenIn: tokenIn})
				require.NoError(t, err)
				got, err := quoteOut(t, sim, tokenIn, tokenOut, res.TokenAmountIn.Amount)
				require.NoError(t, err)
				assert.GreaterOrEqualf(t, got.Cmp(bi(t, c.out)), 0, "tax %d elapsed %d %+v: in %s gives %s",
					tax, elapsed, c, res.TokenAmountIn.Amount, got)
			}
		}
	}
}

// The hook address alone never prices a pool on another chain: the same Robinhood pool built for Arc is refused.
func TestSimulator_WrongChainRefused(t *testing.T) {
	state := poolState{"40987439322977844309085", "1881323003042868532779145197824480", 201513, 202055}
	wrong := rh4
	wrong.chainID = valueobject.ChainIDArc
	_, err := quoteOut(t, wrong.sim(t, state, rh4.extra, rh4.hook), wethRh, rh4.coin, bignumber.TenPowInt(16))
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), ErrNotTracked.Error()))
}

func BenchmarkCalcAmountOut(b *testing.B) {
	pinClock(b, rh4.extra.Launched+3600)
	sim := rh4.sim(b, poolState{"40987439322977844309085", "1881323003042868532779145197824480", 201513, 202055},
		rh4.extra, rh4.hook)
	amountIn := bignumber.TenPowInt(16)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := quoteOut(b, sim, wethRh, rh4.coin, amountIn); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBeforeSwap(b *testing.B) {
	h := hookAt(100, true)
	p := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: bignumber.TenPowInt(18)}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := h.BeforeSwap(p); err != nil {
			b.Fatal(err)
		}
	}
}
