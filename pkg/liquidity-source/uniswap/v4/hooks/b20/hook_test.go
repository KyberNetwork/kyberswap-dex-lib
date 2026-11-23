package b20

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/hooktest"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// reference values pulled from LaunchHook.sol's live poolConfig for a real B20 launch
// on Base (poolId 0x68d39022eee9e18f82fe929f70dd8d2009e442e6d80c6c1ffc170c32b7d3b671),
// so the decay math is checked against the actual on-chain fee schedule, not just its
// own logic restated.
func refExtra() Extra {
	return Extra{
		TokenIsCurrency0:       false,
		BaseFeeBps:             100,
		AntiSnipeStartTotalBps: 9900,
		AntiSnipeWindowSeconds: 16,
		LaunchTime:             1786986263,
	}
}

// totalFeeBps must reproduce LaunchHook.sol's _totalFeeBps exactly: a wrong bps here
// silently mis-prices every swap during the anti-snipe window (first 16s after
// launch for this reference pool) without erroring, so a route built on this quote
// would settle for less than it should on-chain.
func TestTotalFeeBps_Decay(t *testing.T) {
	h := &Hook{Extra: refExtra()}

	cases := []struct {
		name    string
		elapsed int64
		want    int64
	}{
		{"at launch instant, surcharge is maximal", 0, 9900},
		{"halfway through the window, surcharge is halved", 8, 5000},
		{"window boundary, decays to just base fee", 16, 100},
		{"long after the window, stays at base fee", 1_000_000, 100},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			NowFn = func() int64 { return h.LaunchTime + c.elapsed }
			assert.Equal(t, c.want, h.TotalFeeBps())
		})
	}
}

// A misconfigured pool (antiSnipeStartTotalBps above the hard ceiling somehow making
// it through Track) must never let the simulator quote a fee above what the contract
// can ever charge -- MaxTotalFeeBps is the contract's own hard ceiling.
func TestTotalFeeBps_CappedAtMax(t *testing.T) {
	// On-chain _validateFeeConfig rejects antiSnipeStartTotalBps > MAX_TOTAL_FEE_BPS at
	// registration time, but the Go clamp must hold defensively too -- use an
	// out-of-policy value (10_000) so the surcharge math alone would exceed the cap and
	// the assertion can actually fail if the `if total > MaxTotalFeeBps` clamp is removed.
	h := &Hook{BaseFeeBps: 0, AntiSnipeStartTotalBps: 10_000, AntiSnipeWindowSeconds: 10, LaunchTime: 1}
	NowFn = func() int64 { return 1 } // elapsed=0 -> surcharge=10_000, would exceed the cap uncapped
	assert.Equal(t, int64(MaxTotalFeeBps), h.TotalFeeBps())
}

func TestQuoteIsSpecified(t *testing.T) {
	cases := []struct {
		name             string
		tokenIsCurrency0 bool
		zeroForOne       bool
		want             bool
	}{
		{"token=c1, sell c0(quote) for c1(token): quote is specified (buying token)", false, true, true},
		{"token=c1, sell c1(token) for c0(quote): token is specified (selling token)", false, false, false},
		{"token=c0, sell c0(token) for c1(quote): token is specified (selling token)", true, true, false},
		{"token=c0, sell c1(quote) for c0(token): quote is specified (buying token)", true, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := &Hook{TokenIsCurrency0: c.tokenIsCurrency0}
			assert.Equal(t, c.want, h.QuoteIsSpecified(c.zeroForOne))
		})
	}
}

// Buying the token with quote-as-input: LaunchHook.sol charges the fee pre-swap
// (beforeSwap), floor-rounded, reducing what the AMM sees as input.
func TestBeforeSwap_QuoteSpecified_ChargesFeeOnInput(t *testing.T) {
	e := refExtra()
	NowFn = func() int64 { return e.LaunchTime + e.AntiSnipeWindowSeconds } // post-window: flat 100bps
	h := &Hook{Extra: e}

	// tokenIsCurrency0=false -> quote is currency0 -> zeroForOne=true sells quote in.
	amountIn := big.NewInt(1_000_000)
	result, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: u256.FromBig(amountIn)})
	require.NoError(t, err)

	// floor(1_000_000 * 100 / 10_000) = 10_000, matching _feeAmount's floor mulDiv.
	assert.Equal(t, big.NewInt(10_000), u256.ToBig(result.DeltaSpecified))
	assert.Zero(t, u256.ToBig(result.DeltaUnspecified).Sign())
}

// Selling the token for quote: the fee must NOT be charged in BeforeSwap (it's
// deferred to AfterSwap on the realized output) -- charging it here too would
// double-count against AfterSwap's own fee application.
func TestBeforeSwap_TokenSpecified_NoFeeHere(t *testing.T) {
	e := refExtra()
	NowFn = func() int64 { return e.LaunchTime + e.AntiSnipeWindowSeconds }
	h := &Hook{Extra: e}

	result, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: false, AmountSpecified: u256.FromBig(big.NewInt(1_000_000))})
	require.NoError(t, err)
	assert.Zero(t, u256.ToBig(result.DeltaSpecified).Sign())
	assert.Zero(t, u256.ToBig(result.DeltaUnspecified).Sign())
}

// Selling the token for quote: fee lands post-swap on the realized quote output.
func TestAfterSwap_TokenSpecified_ChargesFeeOnOutput(t *testing.T) {
	e := refExtra()
	NowFn = func() int64 { return e.LaunchTime + e.AntiSnipeWindowSeconds }
	h := &Hook{Extra: e}

	amountOut := big.NewInt(1_000_000)
	result, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: false},
		AmountOut:        u256.FromBig(amountOut),
	})
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(10_000), u256.ToBig(result.HookFee))
}

// CalcAmountIn reverses exact-in, so it must invert the quote-side fee on both sides, including the steep
// anti-snipe surcharge. Goldens cover the anti-snipe window (elapsed 2) and its boundary (elapsed 16) for both
// currency orderings, 32 output amounts per direction, captured from the legacy CalcAmountIn.
func TestCalcAmountIn(t *testing.T) {
	e := refExtra()
	for _, c := range []struct {
		elapsed          int64
		tokenIsCurrency0 bool
		want             map[int]map[int]map[string]string
	}{
		{elapsed: 2, tokenIsCurrency0: false, want: map[int]map[int]map[string]string{
			0: {1: {
				"831":          "123138751695668",
				"1145":         "169667713871472",
				"1146":         "169815895457419",
				"3247":         "481145035827117",
				"4169":         "617768295100514",
				"93927":        "13918235214841714",
				"191870":       "28431566968924272",
				"202400":       "29991917207072853",
				"377778":       "55979676376157087",
				"383969":       "56897067479732431",
				"517729":       "76717812764186680",
				"1250463":      "185295369396112725",
				"1843632":      "273191987663001525",
				"16778556":     "2486270071123986997",
				"43538888":     "6451653775514945570",
				"66539098":     "9859857303506630023",
				"111622120":    "16540323031227028861",
				"1076873565":   "159572642348302771027",
				"1306739142":   "193634447475856420589",
				"1427539910":   "211534875512600857910",
				"1750846600":   "259442916451093835389",
				"2715681786":   "402413554032843013653",
				"6106417096":   "904857491267795446551",
				"17749781346":  "2630187622403757047834",
				"25120980578":  "3722462322498930730099",
				"25963900242":  "3847367347583035791208",
				"37922604079":  "5619424959963877555895",
				"65182659378":  "9658858464554762819034",
				"75983070398":  "11259278641560691891185",
				"84501355134":  "12521530124534247542748",
				"134845844248": "19981647846597686333782",
				"144814498516": "21458817179927346508317",
			}},
			1: {0: {
				"65685147139659":               "25402",
				"89122184205396":               "34465",
				"3697252830857373":             "1429762",
				"3702443521168645":             "1431769",
				"7814524627574926":             "3021947",
				"10723646412694352":            "4146931",
				"13884555330005260":            "5369282",
				"126525561528933792":           "48928566",
				"130271090009946176":           "50376995",
				"174450023975122496":           "67461384",
				"192372454839994304":           "74392148",
				"865366241483181440":           "334644858",
				"878313009331863168":           "339651489",
				"8022554640977725440":          "3102393563",
				"25222185074937192448":         "9753644348",
				"32680920598149554176":         "12638004032",
				"39758200691593699328":         "15374851496",
				"711071662062458109952":        "274977782695",
				"1337931841190111477760":       "517390259745",
				"5837781655605370421248":       "2257523516978",
				"6833728823095731421184":       "2642665626376",
				"14565737493903706161152":      "5632708246917",
				"14823146193914356039680":      "5732250755533",
				"16601853675957995438080":      "6420094785024",
				"55418580732857569771520":      "21430967674164",
				"133543907993149924966400":     "51643245774194",
				"296787004746006651731968":     "114773265322640",
				"4582388120476301331005440":    "1772784689907363",
				"4685504228580589857406976":    "1812694034908145",
				"20383374324229171994165248":   "7896982194955245",
				"310929210508767475716325376":  "123715985150103825",
				"2181895907722592702307500032": "1051037357987402429",
			}},
		}},
		{elapsed: 2, tokenIsCurrency0: true, want: map[int]map[int]map[string]string{
			0: {1: {
				"774":          "114544970184788",
				"842":          "124636865115757",
				"1072":         "158721552842615",
				"13288":        "1968901201235080",
				"26977":        "3997352435728546",
				"60590":        "8978173770029649",
				"101310":       "15012125565448876",
				"123107":       "18242022774157990",
				"8127322":      "1204317884112181469",
				"11676015":     "1730168219709972242",
				"50550234":     "7490604768802696059",
				"76524778":     "11339549294223367960",
				"109231061":    "16186012403828968976",
				"127813780":    "18939625894687128613",
				"156521360":    "23193555551118752077",
				"187069847":    "27720273406671470718",
				"202728922":    "30040657205097286691",
				"805313648":    "119332511181810424971",
				"874782346":    "129626480764517773689",
				"965761871":    "143107955033439104447",
				"1072865304":   "158978692709991601625",
				"1180905635":   "174988261253055320985",
				"2452838293":   "363465036045657447241",
				"2611809037":   "387021545111852690510",
				"4838081108":   "716913682029418252095",
				"5594613469":   "829017715393713993761",
				"15170677077":  "2248012366812569411151",
				"54819656182":  "8123254688408331508379",
				"59509562899":  "8818211814955274180935",
				"60431673785":  "8954851534416173681475",
				"90565320165":  "13420098909934876708480",
				"225020073579": "33343804858840977741158",
			}},
			1: {0: {
				"39691526697314":                "15351",
				"146100359986886":               "56499",
				"264223456190624":               "102174",
				"564839105203396":               "218423",
				"1220880864554581":              "472121",
				"987176031781558272":            "381749789",
				"4557752076467273216":           "1762523457",
				"26689868932987674624":          "10321210793",
				"64240872932887166976":          "24842519563",
				"93295930569397485568":          "36078369978",
				"134617798497985200128":         "52057905548",
				"632736634172010921984":         "244684911684",
				"2337434157857558233088":        "903906698710",
				"33515717863209158311936":       "12960832757140",
				"65180508750377220308992":       "25205904786310",
				"79822396557260958466048":       "30868063456748",
				"622880967410591452889088":      "240875432297570",
				"2472682267796129559085056":     "956236643258234",
				"4337989032485914434076672":     "1677626274997465",
				"21872597436002132379566080":    "8460545097986438",
				"30383053619889942942449664":    "11753675147082423",
				"204823286948844209356931072":   "79401767744788551",
				"435077833222798850601779200":   "169129748104920453",
				"519583839449392903637434368":   "202185840968015012",
				"529374763616050039264116736":   "206020095456645985",
				"723537488961301483813339136":   "282244118204198106",
				"1060279466709930765279297536":  "415292975093941276",
				"1440185274917285233514512384":  "566707032150916355",
				"23876485355783061617145544704": "12930802027264524710",
				"32383812986694624503616503808": "20457064166563702204",
				"50898685720645306468641275904": "50413838306404621812",
				"55994233773038556226181398528": "65735531580062433525",
			}},
		}},
		{elapsed: 16, tokenIsCurrency0: false, want: map[int]map[int]map[string]string{
			0: {1: {
				"9071":          "179899340531617",
				"18563":         "368148104559044",
				"30682":         "608496479147894",
				"63128":         "1251977241802082",
				"345502":        "6852120152568905",
				"357266":        "7085428039211785",
				"508397":        "10082712485499194",
				"1444798":       "28653754513770426",
				"2059838":       "40851449399946380",
				"2658938":       "52733016462890712",
				"2793541":       "55402511657911564",
				"3328865":       "66019250109476589",
				"3488385":       "69182908226412456",
				"4781086":       "94820220233870933",
				"5043603":       "100026551965848840",
				"5234112":       "103804795096497483",
				"23412837":      "464331819307875564",
				"23595976":      "467963897943131928",
				"33733769":      "669020261488180542",
				"34594362":      "686087851946210350",
				"299603496":     "5941844483762415934",
				"535608331":     "10622377407724237033",
				"2937028510":    "58248207696436404581",
				"48860161842":   "969012345642687590765",
				"49302581435":   "977786570621980395596",
				"120898745620":  "2397707552337793195010",
				"161657892660":  "3206057695982586034514",
				"194898130810":  "3865290130928017882621",
				"259516674042":  "5146828499651689674775",
				"272024272474":  "5394883729300995713212",
				"454425611534":  "9012333422964288257123",
				"2871002174424": "56938787902596107264808",
			}},
			1: {0: {
				"952769319472548":              "49313",
				"2867964959170797":             "148437",
				"3251828208034025":             "168304",
				"10260623929773582":            "531055",
				"16439938552395668":            "850875",
				"24035590750443520":            "1243999",
				"53806668262736256":            "2784846",
				"361009972388583040":           "18684613",
				"363482814997568512":           "18812599",
				"908687161656179840":           "47030467",
				"2681959854011480064":          "138808853",
				"39099189459188277248":         "2023637156",
				"295957383803555282944":        "15317718052",
				"586371701952995786752":        "30348546536",
				"1803809398777390301184":       "93358861117",
				"3929515074161082368000":       "203377952097",
				"15562714988185376522240":      "805471792077",
				"19956478560246079422464":      "1032877709066",
				"20950288950358731390976":      "1084313880835",
				"108999176016768018677760":     "5641423167147",
				"284905328871373294010368":     "14745753148800",
				"766772269874396136472576":     "39685816222389",
				"813224639867030465413120":     "42090074210111",
				"3070888300877770011967488":    "158944331446153",
				"3384604378631896257527808":    "175182445324354",
				"93774752016337461638594560":   "4858964891979638",
				"298927582680296615609106432":  "15527595562301100",
				"426683405466299657160228864":  "22198218729970562",
				"1070518928098858094666186752": "56133250311727655",
				"4879990073958957976481955840": "268416923103044917",
				"6767802285283726468988272640": "381512627627911777",
				"7930936264181721063495827456": "454038771575326839",
			}},
		}},
		{elapsed: 16, tokenIsCurrency0: true, want: map[int]map[int]map[string]string{
			0: {1: {
				"7232":          "143426638186909",
				"23049":         "457100008614771",
				"51658":         "1024484401540914",
				"57259":         "1135573781134740",
				"475428":        "9428853448670229",
				"498733":        "9891038672796914",
				"983187":        "19498915268091138",
				"1015043":       "20130679667301678",
				"1421505":       "28191786850727050",
				"1846455":       "36619559669593090",
				"1948271":       "38638802540976060",
				"2475231":       "49089666502281907",
				"8366969":       "165936727465115189",
				"15932150":      "315972123476916822",
				"16759253":      "332375536517594858",
				"235533216":     "4671179597633446026",
				"293492419":     "5820647385340004104",
				"374720429":     "7431590548316409975",
				"1013802886":    "20106104121346937425",
				"1215248904":    "24101254130651306051",
				"2866345523":    "56846397227230587745",
				"3511899326":    "69649252858792943933",
				"5515258416":    "109380592242578823849",
				"7964170095":    "157948291195027517612",
				"11171458110":   "221556383027576591852",
				"13178207567":   "261354961533267171961",
				"15481965710":   "307043923420833798592",
				"23194211694":   "459995965978035543999",
				"106175312452":  "2105707105697521407159",
				"197816773332":  "3923173708201223038206",
				"233613234700":  "4633102100470565045071",
				"1863317692009": "36954004148839369838025",
			}},
			1: {0: {
				"1049778080164149":              "54334",
				"1589648019745818":              "82275",
				"5930421156251932":              "306938",
				"41324928548187416":             "2138834",
				"670687052078353792":            "34712414",
				"886506089297531264":            "45882451",
				"1043606866398920448":           "54013438",
				"5056628291642727424":           "261713376",
				"5247665457357799424":           "271600791",
				"6142874312382823424":           "317933667",
				"7745000747940784128":           "400854123",
				"36195986983027523584":          "1873377560",
				"58734414591874318336":          "3039887664",
				"87129834266738688000":          "4509535172",
				"148504996036870438912":         "7686098671",
				"264971468600994201600":         "13713995554",
				"1561575364854420078592":        "80821675092",
				"2667693839066044301312":        "138070497533",
				"3971207254020754243584":        "205535793463",
				"109879074977360042262528":      "5686963701680",
				"343111660173520428597248":      "17758331218152",
				"418371705473681175085056":      "21653563189525",
				"610464866741011412418560":      "31595755173006",
				"1303378251881421096353792":     "67459348004810",
				"3315330073677518384136192":     "171596690401546",
				"11069272047957043153207296":    "572982725010148",
				"12572912536715392077266944":    "650827932022481",
				"29556107463546613067153408":    "1530262307969279",
				"121076148638202470714048512":   "6275577323847946",
				"4933997955258288695148544000":  "271403837128157281",
				"7765602542604354888260386816":  "443132752550099466",
				"54977559085967261870625652736": "8330278880616943368",
			}},
		}},
	} {
		e := e
		e.TokenIsCurrency0 = c.tokenIsCurrency0
		NowFn = func() int64 { return e.LaunchTime + c.elapsed }
		pSim := hooktest.NewPoolSimulator(t, HookAddresses[0], e)
		hooktest.RequireRoundTrip(t, pSim)
		testutil.TestCalcAmountInIPoolSim(t, pSim, c.want)
	}
}

// A pool whose Track() never succeeded (RPC failure, or a factory that constructs the
// Hook before HookExtra is populated) must refuse to quote rather than default to a
// zero-value Extra (BaseFeeBps=0), which would silently price the pool as fee-free.
func TestBeforeAfterSwap_UntrackedPool_Errors(t *testing.T) {
	h := &Hook{} // zero-value Extra: LaunchTime == 0

	_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: u256.FromBig(big.NewInt(1_000_000))})
	assert.ErrorIs(t, err, ErrNotTracked)

	_, err = h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: false},
		AmountOut:        u256.FromBig(big.NewInt(1_000_000)),
	})
	assert.ErrorIs(t, err, ErrNotTracked)
}
