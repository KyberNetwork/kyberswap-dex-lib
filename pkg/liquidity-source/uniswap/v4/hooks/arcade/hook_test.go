package arcade

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/hooktest"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func tokens(n int64) *big.Int { return new(big.Int).Mul(big.NewInt(n), big.NewInt(1e18)) }

func graduatedPump(usdcIsCurrency0 bool) Extra {
	return Extra{Tracked: true, Mode: ModePump, Status: StatusGraduated, UsdcIsCurrency0: usdcIsCurrency0}
}

func TestPumpFeeBps(t *testing.T) {
	cases := []struct {
		name string
		e    Extra
		want int64
	}{
		{"oracle not seeded charges the max", Extra{ObsInit: false}, 100},
		{"at the graduation mcap", Extra{ObsInit: true, EmaTickE3: 50_000_000, GradMcapTick: 50_000}, 100},
		{"below the graduation mcap", Extra{ObsInit: true, EmaTickE3: 49_000_000, GradMcapTick: 50_000}, 100},
		{"half way to the floor", Extra{ObsInit: true, EmaTickE3: (50_000 + 11_513) * 1_000, GradMcapTick: 50_000}, 100 - (70*11_513)/23_026},
		{"one tick short of the floor", Extra{ObsInit: true, EmaTickE3: (50_000 + 23_025) * 1_000, GradMcapTick: 50_000}, 100 - (70*23_025)/23_026},
		{"at the floor", Extra{ObsInit: true, EmaTickE3: (50_000 + 23_026) * 1_000, GradMcapTick: 50_000}, 30},
		{"far above the floor", Extra{ObsInit: true, EmaTickE3: 900_000_000, GradMcapTick: 50_000}, 30},
		// int256(emaTickE3) / 1000 truncates toward zero in Solidity, as in Go:
		// -39_999_500 / 1000 = -39_999, growth = -39_999 - (-50_000) = 10_001.
		{"negative ticks truncate toward zero", Extra{ObsInit: true, EmaTickE3: -39_999_500, GradMcapTick: -50_000}, 100 - (70*10_001)/23_026},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.e.PumpFeeBps())
		})
	}
}

// Buying with USDC exact-in: the fee comes off the specified USDC input (beforeSwap).
func TestPump_BuyTakesFeeOnInput(t *testing.T) {
	for _, usdc0 := range []bool{true, false} {
		h := &Hook{Extra: graduatedPump(usdc0)}
		zeroForOne := usdc0 // USDC in
		in := big.NewInt(1_234_567_891)

		before, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: zeroForOne, AmountSpecified: u256.FromBig(in)})
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(12_345_678), u256.ToBig(before.DeltaSpecified)) // floor(1_234_567_891 * 100 / 10_000)
		assert.Zero(t, u256.ToBig(before.DeltaUnspecified).Sign())

		after, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
			BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: zeroForOne},
			AmountOut:        u256.FromBig(tokens(1_000)),
		})
		require.NoError(t, err)
		assert.Zero(t, u256.ToBig(after.HookFee).Sign())
	}
}

// Selling the token exact-in: nothing before the swap, the fee comes off the USDC output.
func TestPump_SellTakesFeeOnOutput(t *testing.T) {
	for _, usdc0 := range []bool{true, false} {
		e := graduatedPump(usdc0)
		e.ObsInit, e.EmaTickE3, e.GradMcapTick = true, 60_000_000, 50_000 // growth 10_000 ticks
		h := &Hook{Extra: e}
		zeroForOne := !usdc0 // token in

		before, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: zeroForOne, AmountSpecified: u256.FromBig(tokens(5))})
		require.NoError(t, err)
		assert.Zero(t, u256.ToBig(before.DeltaSpecified).Sign())

		out := big.NewInt(987_654_321)
		after, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
			BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: zeroForOne},
			AmountOut:        u256.FromBig(out),
		})
		require.NoError(t, err)
		feeBps := int64(100 - (70*10_000)/23_026) // 70
		want := new(big.Int).Div(new(big.Int).Mul(out, big.NewInt(feeBps)), big.NewInt(10_000))
		assert.Equal(t, want, u256.ToBig(after.HookFee))
	}
}

// CLANKER / RWA: the fee is the pool's native LP fee, the hook takes nothing.
func TestDirectLaunch_NoHookFee(t *testing.T) {
	NowFn = func() int64 { return 2_000_000_000 }
	for _, mode := range []uint8{ModeClanker, ModeRwa} {
		h := &Hook{Tracked: true, Mode: mode, Status: StatusGraduated, QuoteIsCurrency0: true, LaunchedAt: 1_000, BuyCapEnabled: true}
		before, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: u256.FromBig(big.NewInt(1e9))})
		require.NoError(t, err)
		assert.Zero(t, u256.ToBig(before.DeltaSpecified).Sign())
		after, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
			BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true},
			AmountOut:        u256.FromBig(tokens(500_000_000)), // far above any cap, but long after the window
		})
		require.NoError(t, err)
		assert.Zero(t, u256.ToBig(after.HookFee).Sign())
	}
}

func TestPerTxMaxBuyTokens(t *testing.T) {
	e := Extra{LaunchedAt: 1_000}
	cases := []struct {
		elapsed int64
		want    *big.Int
	}{
		{0, tokens(10_000_000)},   // 1%
		{59, tokens(10_000_000)},  // still the first minute
		{60, tokens(20_000_000)},  // 2%
		{240, tokens(50_000_000)}, // 5%
		{299, tokens(50_000_000)}, // last second of the window
		{300, nil},                // uncapped
		{10_000, nil},
	}
	for _, c := range cases {
		got := e.PerTxMaxBuyTokens(1_000 + c.elapsed)
		if c.want == nil {
			assert.Nil(t, got, "elapsed %d", c.elapsed)
		} else {
			assert.Equal(t, c.want, got, "elapsed %d", c.elapsed)
		}
	}
}

func TestDirectLaunch_BuyCap(t *testing.T) {
	h := &Hook{Tracked: true, Mode: ModeClanker, Status: StatusGraduated, QuoteIsCurrency0: false, LaunchedAt: 1_000, BuyCapEnabled: true}
	// cap = 1% = 10M tokens; the quote is currency1, so a buy is oneForZero.
	NowFn = func() int64 { return 1_000 + 30 }
	buy := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: false}

	_, err := h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: buy, AmountOut: u256.FromBig(tokens(10_000_000))})
	require.NoError(t, err, "exactly at the cap passes (the contract reverts only above it)")

	_, err = h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: buy, AmountOut: u256.FromBig(new(big.Int).Add(tokens(10_000_000), big.NewInt(1)))})
	assert.ErrorIs(t, err, ErrBuyExceedsCap)

	sell := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true}
	_, err = h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: sell, AmountOut: u256.FromBig(tokens(900_000_000))})
	require.NoError(t, err, "sells are never capped")

	h.BuyCapEnabled = false
	_, err = h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: buy, AmountOut: u256.FromBig(tokens(900_000_000))})
	require.NoError(t, err, "owner-disabled cap")
}

func TestRefusesUntradablePools(t *testing.T) {
	params := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: u256.FromBig(big.NewInt(1_000))}

	_, err := (&Hook{}).BeforeSwap(params)
	assert.ErrorIs(t, err, ErrNotTracked)

	for _, status := range []uint8{StatusCurving, StatusGraduationStarted} {
		_, err = (&Hook{Tracked: true, Mode: ModePump, Status: status}).BeforeSwap(params)
		assert.ErrorIs(t, err, ErrNotGraduated)
	}

	_, err = (&Hook{Tracked: true, Mode: 2, Status: StatusGraduated}).BeforeSwap(params)
	assert.ErrorIs(t, err, ErrUnknownMode)
}

// CalcAmountIn reverses exact-in, so a v1 PUMP pool must invert its USDC fee on whichever side USDC is.
// Goldens cover max fee (emaTickE3 0) and floor fee (900M) for both USDC orderings, 32 output amounts per
// direction, captured from the legacy CalcAmountIn.
func TestPumpV1_CalcAmountIn(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		usdcIsCurrency0 bool
		emaTickE3       int64
		want            map[int]map[int]map[string]string
	}{
		{usdcIsCurrency0: false, emaTickE3: 0, want: map[int]map[int]map[string]string{
			0: {1: {
				"6739":          "133648887922345",
				"8585":          "170246732388851",
				"17463":         "346324773539455",
				"17528":         "347620620029553",
				"125719":        "2493287053632689",
				"141174":        "2799794000688441",
				"259396":        "5144431762349493",
				"477984":        "9479548531455145",
				"598165":        "11863022417776606",
				"1140055":       "22609967652622051",
				"1721168":       "34134813422061153",
				"4354039":       "86350866491546838",
				"10527035":      "208775937649383398",
				"22362347":      "443498117372327049",
				"24926655":      "494354395596156942",
				"29447618":      "584015757018208055",
				"37972267":      "753079676048429302",
				"53162332":      "1054334518294445653",
				"108753935":     "2156847212599959991",
				"307444926":     "6097358542958956587",
				"969580604":     "19229071884948217927",
				"6092814187":    "120834886431038786588",
				"78677202956":   "1560354666403238841348",
				"86215048075":   "1709847932621159465271",
				"116256682305":  "2305644475595455202776",
				"127366019014":  "2525968856737894037757",
				"131842013362":  "2614738394847395223742",
				"214757040008":  "4259139213524747071925",
				"252143865407":  "5000608287731411546808",
				"655326846070":  "12996680150111462891347",
				"1924496375738": "38167322936255066640908",
				"4539030378711": "90019782695245756005579",
			}},
			1: {0: {
				"892956615701758":               "46217",
				"1265709564011769":              "65510",
				"1282267370113639":              "66366",
				"1303108187270766":              "67445",
				"19822500362339628":             "1025944",
				"32348909003463508":             "1674267",
				"53195793918053680":             "2753229",
				"99307476978584432":             "5139809",
				"570916353765610432":            "29548632",
				"1509475025111956224":           "78125143",
				"3570088163445755392":           "184775264",
				"3982405788523873280":           "206115381",
				"9523795400373604352":           "492918307",
				"119532156002086682624":         "6186565905",
				"237854412922539016192":         "12310511685",
				"413772269868596330496":         "21415404118",
				"1543813801693473406976":        "79902398735",
				"3877643682544811507712":        "200693270129",
				"10149364311318499164160":       "525295626186",
				"570053230881592196464640":      "29504160816938",
				"1095873431287080771649536":     "56719316395210",
				"1863594461914820553736192":     "96455277391342",
				"17469805836559481737052160":    "904365198459176",
				"28602510275774818094678016":    "1480873061692776",
				"30132678655025930542514176":    "1560124957494880",
				"36919779010415449478266880":    "1911683766296534",
				"57220074690637756800958464":    "2963541580110755",
				"57864395925179362707505152":    "2996935401278302",
				"179599373757749267261292544":   "9315471568312113",
				"14765347855190654596331601920": "928368418732963259",
				"17231501503662964063980224512": "1123747210971760518",
				"73783313676703206402005401600": "32819231879017674444",
			}},
		}},
		{usdcIsCurrency0: false, emaTickE3: 900000000, want: map[int]map[int]map[string]string{
			0: {1: {
				"6765":          "133216939190909",
				"24876":         "489869215939290",
				"145882":        "2872852251348394",
				"148240":        "2919306382172120",
				"217660":        "4286385090115496",
				"682596":        "13442423598698669",
				"754858":        "14865498579304841",
				"931507":        "18344257203278473",
				"1314191":       "25880507317453884",
				"2187625":       "43081140784564747",
				"6263152":       "123340959702065487",
				"6378322":       "125609024720597528",
				"9973241":       "196404177706593425",
				"10686849":      "210457357285068121",
				"15185878":      "299057243780033100",
				"25054068":      "493392661575967960",
				"95221074":      "1875199645863460525",
				"325693891":     "6413927580465850417",
				"2924657183":    "57595613357820551445",
				"5405263003":    "106446471756280292994",
				"7660933854":    "150867659753888736830",
				"16338779240":   "321761477032610668165",
				"19198451829":   "378077341677779475306",
				"30299952993":   "596700495057190143533",
				"30669740307":   "603982759651716317942",
				"1525776672905": "30047307876476390307556",
				"1937345882480": "38152394686217299480611",
				"2277968561462": "44860323283839485705850",
				"2303198541032": "45357180734851872473209",
				"2953017596275": "58154158622131312522647",
				"3173332950340": "62492860708979307874142",
				"4915874372259": "96808996437483046702847",
			}},
			1: {0: {
				"2587954558311538":              "133004",
				"3085802197401494":              "158589",
				"3984300242410934":              "204766",
				"46589525044651664":             "2394381",
				"104669242623033840":            "5379278",
				"479314059123748672":            "24633442",
				"1508824270759707136":           "77543177",
				"2171775680748110848":           "111614314",
				"2218735612034480640":           "114027731",
				"3709440610391119360":           "190639702",
				"6787418217594443776":           "348826553",
				"12642882632836440064":          "649757099",
				"25967083722949713920":          "1334529274",
				"55993334758668419072":          "2877671795",
				"78163315233286029312":          "4017056112",
				"81524123672456265728":          "4189778521",
				"152194471952509370368":         "7821747741",
				"163905490897204477952":         "8423613465",
				"9531659862215755825152":        "489861730240",
				"72091057589856199770112":       "3704987082506",
				"154343398987668333264896":      "7932202438700",
				"1175422254395723989647360":     "60409459370918",
				"1178190880925562720747520":     "60551751718395",
				"2254892390697529606406144":     "115889069482572",
				"25590874859834640202465280":    "1315597934415853",
				"25630967792628956494036992":    "1317659699629324",
				"46286999388800016699097088":    "2380152416849808",
				"100292788969877041890459648":   "5160557129951400",
				"129756754889388175732506624":   "6678982669173211",
				"256334180996753971923124224":   "13214378467767211",
				"4400692608182933675774574592":  "238748156356538464",
				"53828348418640892671790415872": "7785196755786090419",
			}},
		}},
		{usdcIsCurrency0: true, emaTickE3: 0, want: map[int]map[int]map[string]string{
			0: {1: {
				"10311":         "204491467233306",
				"11915":         "236302573187406",
				"57553":         "1141411833041497",
				"65810":         "1305167632137734",
				"246665":        "4891949156429971",
				"290726":        "5765782784099878",
				"2584783":       "51262347783956441",
				"3666982":       "72724908281022465",
				"12832788":      "254504475421714054",
				"22740026":      "450988389134045433",
				"23543777":      "466928668567209315",
				"37749103":      "748653812149704213",
				"46431344":      "920843143978959581",
				"56348536":      "1117524469008910808",
				"130761987":     "2593318841349678371",
				"287639617":     "5704572524674964959",
				"299096868":     "5931796854720933860",
				"740822330":     "14692255378134085759",
				"3598274134":    "71362269186293253867",
				"8915579505":    "176816985501171397701",
				"15790197448":   "313156886342335973434",
				"18420544736":   "365322881921482891695",
				"27583396105":   "547043853714752768175",
				"30056514884":   "596091637096518071510",
				"66615839880":   "1321149359605435461927",
				"75117113853":   "1489749693749363686738",
				"172626668136":  "3423594420449727884472",
				"179003723263":  "3550066485882236387468",
				"362986832514":  "7198886263457596994956",
				"412470879001":  "8180271881484041410943",
				"2234333031742": "44312120953036151790885",
				"3308134902369": "65608174225819687437424",
			}},
			1: {0: {
				"3666454542924470":              "189764",
				"90285827002444048":             "4672879",
				"146241041262996288":            "7568925",
				"179162741556931456":            "9272837",
				"1201205685321899520":           "62170202",
				"54423984221810106368":          "2816794880",
				"116075672140442042368":         "6007670402",
				"204302777668937285632":         "10573996505",
				"374567214490011041792":         "19386287696",
				"384472047830932979712":         "19898927196",
				"802596934970141310976":         "41539607749",
				"1349146691225952976896":        "69827110317",
				"2290971455268551917568":        "118572664784",
				"2600785345513459286016":        "134607547987",
				"4279168726180291936256":        "221474802734",
				"9759528409994845224960":        "505119080593",
				"14858723909824684228608":       "769035665228",
				"19703133971984060252160":       "1019765473883",
				"35976949555206970933248":       "1862041758781",
				"45067390629183940984832":       "2332531623070",
				"51781857578589288923136":       "2680049331186",
				"60030086855971682910208":       "3106949377747",
				"821937381556451004121088":      "42541024135474",
				"834388453090686958305280":      "43185460917458",
				"1077856811255876731011072":     "55786822219727",
				"1472194570994236816097280":     "76196993644947",
				"2097790573728755093602304":     "108577050814752",
				"3576654819044916440596480":     "185123140825813",
				"7731354015332226561998848":     "400185243503537",
				"644080881007149567294046208":   "33597145068647157",
				"3366456585469968167934623744":  "181632924426204347",
				"21177228838737752942390018048": "1473562702307177719",
			}},
		}},
		{usdcIsCurrency0: true, emaTickE3: 900000000, want: map[int]map[int]map[string]string{
			0: {1: {
				"26462":         "521119237231357",
				"41903":         "825200642272077",
				"71184":         "1401834773454786",
				"72486":         "1427475210598336",
				"110383":        "2173785229709313",
				"130760":        "2575071855595360",
				"136509":        "2688287579911982",
				"218133":        "4295718484974561",
				"337245":        "6641404924712056",
				"1610928":       "31724192063246381",
				"8802690":       "173352395782919507",
				"15525879":      "305752937032778138",
				"23775880":      "468221164196371886",
				"32226033":      "634631007925599819",
				"33793943":      "665508041522839906",
				"150610185":     "2965983852652905838",
				"185197468":     "3647115230916477360",
				"186995899":     "3682531941322781072",
				"303363884":     "5974180175492470693",
				"490585336":     "9661153958746101087",
				"918319484":     "18084572179568395984",
				"2382159965":    "46912152682916193465",
				"4212455524":    "82956375600357827377",
				"10501187248":   "206801099718649400522",
				"11664688662":   "229714068215922274771",
				"11789791264":   "232177728289668237940",
				"24768572584":   "487770377291429829292",
				"417599585176":  "8223837901276457616183",
				"576309833648":  "11349337972111463391729",
				"800562438882":  "15765571978540093734555",
				"938363531101":  "18479305986665386087757",
				"2454380614251": "48334430095442937786345",
			}},
			1: {0: {
				"227767592961356":               "11707",
				"649598438531682":               "33386",
				"784664093674059":               "40327",
				"2935833776201419":              "150882",
				"4586829638704537":              "235733",
				"11773406453841302":             "605073",
				"11808096539105868":             "606856",
				"12494005474434248":             "642107",
				"17328102761223648":             "890546",
				"18204122575561264":             "935567",
				"25831903536832120":             "1327583",
				"28999960527944264":             "1490400",
				"179552317561992096":            "9227754",
				"2155925265836238592":           "110799713",
				"6037843891782361088":           "310303595",
				"14079495961205587968":          "723589131",
				"14624312187641049088":          "751588933",
				"235453827735640047616":         "12100705258",
				"302624432748035506176":         "15552811788",
				"390357120848300736512":         "20061667797",
				"646415800298532241408":         "33221320660",
				"1239005879772755001344":        "63676370389",
				"2081113332464581869568":        "106954895761",
				"6448698829450509287424":        "331418734634",
				"11920465242297055838208":       "612629907366",
				"40949974571854327709696":       "2104547714100",
				"57657826160402478661632":       "2963217239277",
				"61961203555710267293696":       "3184381518703",
				"116722388655428645945344":      "5998735344405",
				"14789064719191911694336000":    "760191124857839",
				"12412414104970092220002598912": "749692187352182559",
				"34658219390174887404296470528": "3051672640353862922",
			}},
		}},
	} {
		e := graduatedPump(c.usdcIsCurrency0)
		e.ObsInit, e.EmaTickE3, e.GradMcapTick = c.emaTickE3 != 0, c.emaTickE3, 50_000
		pSim := hooktest.NewPoolSimulator(t, HookV1, e)
		hooktest.RequireRoundTrip(t, pSim)
		testutil.TestCalcAmountInIPoolSim(t, pSim, c.want)
	}
}

func graduatedPumpV2(usdcIsCurrency0 bool) Extra {
	e := graduatedPump(usdcIsCurrency0)
	e.NoSwapDelta = true
	return e
}

// The generation comes from the hook address: v1 carries both RETURNS_DELTA permission
// bits (0x3ECE), v2 neither (0x3EC2), and every registered address has an entry.
func TestGenerations(t *testing.T) {
	permissions := func(a common.Address) uint16 { return (uint16(a[18])<<8 | uint16(a[19])) & 0x3FFF }
	assert.Equal(t, uint16(0x3ECE), permissions(HookV1))
	assert.Equal(t, uint16(0x3EC2), permissions(HookV2))

	assert.Equal(t, []common.Address{HookV1, HookV2}, HookAddresses)
	assert.Len(t, Generations, len(HookAddresses))
	assert.False(t, Generations[HookV1].NoSwapDelta)
	assert.True(t, Generations[HookV2].NoSwapDelta)

	// The factory trusts the address over whatever extra was persisted.
	for _, c := range []struct {
		hook  common.Address
		extra string
		want  bool
	}{
		{HookV1, `{"tr":true,"m":0,"s":2}`, false},
		{HookV1, `{"tr":true,"m":0,"s":2,"nd":true}`, false},
		{HookV2, `{"tr":true,"m":0,"s":2}`, true},
		{HookV2, `{"tr":true,"m":0,"s":2,"nd":true}`, true},
		{HookV2, ``, true}, // not tracked yet
	} {
		h, ok := uniswapv4.GetHook(c.hook, &uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(c.extra)})
		require.True(t, ok)
		assert.Equal(t, c.want, h.(*Hook).NoSwapDelta, "%s %s", c.hook, c.extra)
		assert.Equal(t, "uniswap-v4-arcade", h.GetExchange())
	}
}

// v2: a graduated PUMP pool pays its fee as the pool's static LP fee. The hook takes
// nothing on either side, for both currency orderings, whatever an oracle would say.
func TestPumpV2_NoHookFee(t *testing.T) {
	for _, usdc0 := range []bool{true, false} {
		e := graduatedPumpV2(usdc0)
		e.ObsInit, e.EmaTickE3, e.GradMcapTick = true, 60_000_000, 50_000 // never read on v2
		h := &Hook{Extra: e}
		for _, zeroForOne := range []bool{true, false} { // a buy and a sell
			for _, calcOut := range []bool{true, false} { // exact-in and exact-out
				swap := &uniswapv4.BeforeSwapParams{CalcOut: calcOut, ZeroForOne: zeroForOne, AmountSpecified: u256.FromBig(big.NewInt(1_234_567_891))}
				before, err := h.BeforeSwap(swap)
				require.NoError(t, err)
				assert.Zero(t, u256.ToBig(before.DeltaSpecified).Sign())
				assert.Zero(t, u256.ToBig(before.DeltaUnspecified).Sign())
				assert.Zero(t, before.SwapFee, "no fee override: the PoolKey fee stands")

				after, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
					BeforeSwapParams: swap, AmountIn: u256.FromBig(big.NewInt(1_234_567_891)), AmountOut: u256.FromBig(big.NewInt(987_654_321)),
				})
				require.NoError(t, err)
				assert.Zero(t, u256.ToBig(after.HookFee).Sign())
			}
		}
	}
}

// v2 CLANKER / RWA behave as on v1: no hook fee, and the launch-window buy cap, which
// exact-out meets too (the token output is then the specified amount).
func TestDirectLaunchV2_BuyCap(t *testing.T) {
	orig := NowFn
	defer func() { NowFn = orig }()
	NowFn = func() int64 { return 1_000 + 30 } // first minute: cap = 1% = 10M tokens

	for _, mode := range []uint8{ModeClanker, ModeRwa} {
		h := &Hook{Tracked: true, NoSwapDelta: true, Mode: mode, Status: StatusGraduated, QuoteIsCurrency0: true, LaunchedAt: 1_000, BuyCapEnabled: true}
		for _, calcOut := range []bool{true, false} {
			buy := &uniswapv4.BeforeSwapParams{CalcOut: calcOut, ZeroForOne: true}
			after, err := h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: buy, AmountOut: u256.FromBig(tokens(10_000_000))})
			require.NoError(t, err)
			assert.Zero(t, u256.ToBig(after.HookFee).Sign())

			_, err = h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: buy, AmountOut: u256.FromBig(new(big.Int).Add(tokens(10_000_000), big.NewInt(1)))})
			assert.ErrorIs(t, err, ErrBuyExceedsCap)

			sell := &uniswapv4.BeforeSwapParams{CalcOut: calcOut, ZeroForOne: false}
			_, err = h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: sell, AmountOut: u256.FromBig(tokens(900_000_000))})
			require.NoError(t, err, "sells are never capped")
		}
	}
}

// v2 keeps the curve guard (beforeSwap reverts while Curving or GraduationStarted), and
// both generations quote CalcIn.
func TestV2_RefusalsAndExactOut(t *testing.T) {
	params := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: u256.FromBig(big.NewInt(1_000))}

	_, err := (&Hook{NoSwapDelta: true}).BeforeSwap(params)
	assert.ErrorIs(t, err, ErrNotTracked)

	for _, status := range []uint8{StatusCurving, StatusGraduationStarted} {
		_, err = (&Hook{Tracked: true, NoSwapDelta: true, Mode: ModePump, Status: status}).BeforeSwap(params)
		assert.ErrorIs(t, err, ErrNotGraduated)
	}

	_, err = (&Hook{Tracked: true, NoSwapDelta: true, Mode: 2, Status: StatusGraduated}).BeforeSwap(params)
	assert.ErrorIs(t, err, ErrUnknownMode)

	exactOut := &uniswapv4.BeforeSwapParams{CalcOut: false, ZeroForOne: true, AmountSpecified: u256.FromBig(big.NewInt(1))}
	_, err = (&Hook{Extra: graduatedPumpV2(true)}).BeforeSwap(exactOut)
	require.NoError(t, err)
	for _, v1 := range []Extra{graduatedPump(true), {Tracked: true, Mode: ModeClanker, Status: StatusGraduated}} {
		_, err = (&Hook{Extra: v1}).BeforeSwap(exactOut)
		require.NoError(t, err)
	}
}
