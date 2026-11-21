package dualpool

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
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
	testutil.TestCalcAmtOutIPoolSim(t, poolSim, map[int]map[int]map[string]string{
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
	var in uint256.Int
	in.SetUint64(1_000_000_000)
	_, err := poolSim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: entityPool.Tokens[0].Address, Amount: in},
		TokenOut:      entityPool.Tokens[1].Address,
	})
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
}

func TestPoolSimulator_UpdateBalanceThenQuoteAgain(t *testing.T) {
	t.Parallel()
	var in uint256.Int
	in.SetUint64(100_000_000)
	params := pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: entityPool.Tokens[0].Address, Amount: in},
		TokenOut:      entityPool.Tokens[1].Address,
	}
	sim := poolSim.CloneState()
	res, err := sim.CalcAmtOut(params)
	require.NoError(t, err)
	sim.UpdateState(pool.UpdateStateParams{
		TokenAmountIn:  params.TokenAmountIn,
		TokenAmountOut: *res.TokenAmountOut,
		Fee:            *res.Fee,
		SwapInfo:       res.SwapInfo,
		SwapLimit:      params.Limit,
	})
	again, err := sim.CalcAmtOut(params)
	require.NoError(t, err)
	require.Equal(t, -1, again.TokenAmountOut.Amount.Cmp(&res.TokenAmountOut.Amount), "second buy must be worse")
	// the original simulator is untouched
	orig, err := poolSim.CalcAmtOut(params)
	require.NoError(t, err)
	require.Equal(t, "421126678297991910", orig.TokenAmountOut.Amount.String())
}

// CalcAmountIn reverses exact-in: the min amountIn whose exact-in quote reaches amountOut. The quoter's
// exact-in answers in the header therefore need at most their own amountIn.
func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		idxIn               int
		maxAmountIn, amtOut string
	}{
		{0, "100000000", "421126678297991910"},
		{1, "100000000000000000", "22929103"},
		{1, "500000000000000000", "111778632"},
	} {
		tokenIn, tokenOut := entityPool.Tokens[c.idxIn].Address, entityPool.Tokens[1-c.idxIn].Address
		var amtOut uint256.Int
		require.NoError(t, amtOut.SetFromDecimal(c.amtOut))
		res, err := poolSim.CalcAmtIn(pool.CalcAmtInParams{
			TokenAmountOut: pool.TokenAmt{Token: tokenOut, Amount: amtOut},
			TokenIn:        tokenIn,
		})
		require.NoError(t, err)
		amountIn := res.TokenAmountIn.Amount
		var maxAmountIn uint256.Int
		require.NoError(t, maxAmountIn.SetFromDecimal(c.maxAmountIn))
		require.LessOrEqual(t, amountIn.Cmp(&maxAmountIn), 0, "amountIn %s", amountIn.String())

		for delta, reaches := range map[int64]bool{0: true, -1: false} {
			var in uint256.Int
			if delta < 0 {
				in.SubUint64(&amountIn, uint64(-delta))
			} else {
				in.AddUint64(&amountIn, uint64(delta))
			}
			out, err := poolSim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: in},
				TokenOut:      tokenOut,
			})
			require.NoError(t, err)
			require.Equal(t, reaches, out.TokenAmountOut.Amount.Cmp(&amtOut) >= 0,
				"amountIn %s%+d", amountIn.String(), delta)
		}
	}
	// Goldens from the legacy CalcAmountIn: 32 output amounts per direction on the snapshot pool.
	testutil.TestCalcAmountInIPoolSim(t, poolSim, map[int]map[int]map[string]string{
		0: {1: {
			"49214401217351":     "11369",
			"54785799092017":     "12657",
			"75045139065087":     "17336",
			"75168487399939":     "17365",
			"179972458357145":    "41575",
			"189460696503287":    "43767",
			"248083285430506":    "57309",
			"325256228355389":    "75137",
			"359244086121927":    "82989",
			"478092856592322":    "110445",
			"573834569801096":    "132563",
			"865761370064812":    "200005",
			"1280265016072430":   "295768",
			"1787708208554135":   "413012",
			"1875809170527040":   "433368",
			"2019598536282332":   "466593",
			"2260443816616580":   "522244",
			"2623925175402111":   "606235",
			"3603246898840580":   "832551",
			"5710681615214513":   "1319664",
			"9500094789319838":   "2195885",
			"21451042842477236":  "4962101",
			"66646634217396176":  "15462020",
			"69404171295391384":  "16104648",
			"79235377783142000":  "18397625",
			"82505486651593168":  "19160977",
			"83777977882149376":  "19458107",
			"103998247374233072": "24186173",
			"111663834869394032": "25981853",
			"170500867694868608": "39824393",
			"274820070357980544": "64630700",
			"401956090311477824": "95326498",
		}},
		1: {0: {
			"31564":    "136777655381483",
			"33902":    "146909108620317",
			"33992":    "147299113500817",
			"34104":    "147784452935078",
			"59555":    "258074294399283",
			"65839":    "285305739703160",
			"72899":    "315900061992600",
			"82840":    "358979325994037",
			"118524":   "513617699446573",
			"144638":   "626785961785525",
			"172523":   "747630897454382",
			"176428":   "764554105957562",
			"210920":   "914034663109585",
			"354097":   "1534562355922030",
			"407682":   "1766811992392077",
			"456631":   "1978974229251572",
			"597965":   "2591598254911056",
			"722086":   "3129651040818616",
			"879362":   "3811481276017363",
			"2006577":  "8699992510422714",
			"2293964":  "9946827132228336",
			"2557473":  "11090242789301018",
			"4044465":  "17545745925250967",
			"4474159":  "19412184748917751",
			"4522169":  "19620751108873379",
			"4714268":  "20455328965897764",
			"5156856":  "22378502943585342",
			"5168695":  "22429953399236015",
			"6518075":  "28296385663281813",
			"16080282": "69995522661347957",
			"16597067": "72255514460438336",
			"22355711": "97483550803701506",
		}},
	})
}
