package thogprop

import (
	"testing"

	"github.com/holiman/uint256"
)

// stateBlock107066625 is the live makerSnapshot() state at Monad block
// 107066625, captured while integrating this package. Used to verify
// ExactQuote against four independently-called makerQuoteExactInput()
// results at that exact block -- see the fixtures below.
func stateBlock107066625() *State {
	u := func(s string) uint256.Int {
		return *uint256.MustFromDecimal(s)
	}
	return &State{
		V1:             u("2994239055849461445655022230385668861908425422835416817831970788550610092030"),
		V2:             u("1124588846958885914322232869647557697816726"),
		V3:             u("853655712402932"),
		R1:             u("2014949657059124526062761958401416449565747983888532651381762"),
		R2:             u("294984522595088000014157334"),
		R3:             u("43066045109633176"),
		I1:             u("26959946667149500150766263744785391798464467030925802040175863822145"),
		I2:             u("112083"),
		P1:             u("57896044618658097711785492504343953926634992332820282019728792003956564819968"),
		P2:             u("57896044618658097711785492504343953926634992332820282019728792003956564819968"),
		GloballyPaused: false,
		RiskV3Ready:    true,
		PairRiskReady:  true,
		Balances: []uint256.Int{
			u("36406748694"),              // USDC
			u("20581810"),                 // AUSD
			u("2227929976"),               // USDT0
			u("171154752028809148389158"), // WMON
			u("2698350120294236217"),      // WETH
			u("895623"),                   // WBTC
			u("7523178"),                  // cbBTC
			u("1846142"),                  // XAUt0
		},
	}
}

func tok(idx int) tokenMeta { return tokenTable[idx] }

func TestExactQuote_LiveParityFixtures(t *testing.T) {
	const executionBlock = 107066625 // age=0, matches the live call exactly

	tests := []struct {
		name        string
		tokenInIdx  int
		tokenOutIdx int
		amountIn    string
		wantOut     string
	}{
		{"USDC->WMON cross-category with penalty", 0, 3, "1000000000", "37907937342479108369049"},
		{"USDC->AUSD same-category, penalty short-circuits", 0, 1, "10000000", "9998074"},
		{"XAUt0->USDC index-7 sell side", 7, 0, "1000000", "4348564229"},
		{"USDC->XAUt0 index-7 buy side", 0, 7, "1000000000", "229839"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := stateBlock107066625()
			amountIn := uint256.MustFromDecimal(tt.amountIn)
			wantOut := uint256.MustFromDecimal(tt.wantOut)

			gotOut, postedBlock, err := ExactQuote(s, tok(tt.tokenInIdx), tok(tt.tokenOutIdx), amountIn, executionBlock, false)
			if err != nil {
				t.Fatalf("ExactQuote() error = %v", err)
			}
			if gotOut.Cmp(wantOut) != 0 {
				diff := new(uint256.Int).Sub(&gotOut, wantOut)
				t.Errorf("amountOut = %s, want %s (diff %s)", gotOut.Dec(), wantOut.Dec(), diff.Dec())
			}
			if postedBlock != executionBlock {
				t.Errorf("postedBlock = %d, want %d", postedBlock, executionBlock)
			}
		})
	}
}

func TestExactQuote_FastLaneHaircut(t *testing.T) {
	s := stateBlock107066625()
	amountIn := uint256.NewInt(1000000000)

	base, _, err := ExactQuote(s, tok(0), tok(3), amountIn, 107066625, false)
	if err != nil {
		t.Fatalf("ExactQuote(fastLaneHot=false) error = %v", err)
	}
	hot, _, err := ExactQuote(s, tok(0), tok(3), amountIn, 107066625, true)
	if err != nil {
		t.Fatalf("ExactQuote(fastLaneHot=true) error = %v", err)
	}

	want := new(uint256.Int).Mul(&base, uint256.NewInt(9990))
	want.Div(want, uint256.NewInt(10000))
	if hot.Cmp(want) != 0 {
		t.Errorf("fastLaneHot amountOut = %s, want %s (9990/10000 of %s)", hot.Dec(), want.Dec(), base.Dec())
	}
	if hot.Cmp(&base) >= 0 {
		t.Errorf("fastLaneHot amountOut %s should be strictly less than cold amountOut %s", hot.Dec(), base.Dec())
	}
}

func TestExactQuote_SameTokenRejected(t *testing.T) {
	s := stateBlock107066625()
	if _, _, err := ExactQuote(s, tok(0), tok(0), uint256.NewInt(1000000), 107066625, false); err == nil {
		t.Error("expected an error quoting tokenIn == tokenOut, got nil")
	}
}

func TestExactQuote_MaxAgeExceededReturnsSentinel(t *testing.T) {
	s := stateBlock107066625()
	maxAgeBits := bits(&s.R1, 198, 8)
	maxAge := maxAgeBits.Uint64()
	_, _, err := ExactQuote(s, tok(0), tok(3), uint256.NewInt(1000000000), 107066625+maxAge+1, false)
	if err != ErrStalePrice {
		t.Errorf("got err = %v, want ErrStalePrice", err)
	}
}

func TestMaxFlooredInput(t *testing.T) {
	tests := []struct {
		maximum, divisor, multiplier, want string
	}{
		// maximum saturated at maxU256, large divisor -> saturates again
		{"115792089237316195423570985008687907853269984665640564039457584007913129639935", "1000000000000000000000000000000", "1000000000000000000", "115792089237316195423570985008687907853269984665640564039457584007913129639935"},
		// maximum saturated, tiny divisor -> does NOT saturate, needs the wide-multiply path
		{"115792089237316195423570985008687907853269984665640564039457584007913129639935", "1", "1000000000000000000", "115792089237316195423570985008687907853269984665640564039457"},
		// maxRiskNotionalUsdWad as maximum, realistic WAD-scale divisor/multiplier
		{"664619068552888026590626425199190042", "1000000000000000000", "1000000000000000000000000000000", "664619068552888026590626"},
		{"100", "7", "3", "235"},
		{"0", "5", "3", "1"},
		{"115792089237316195423570985008687907853269984665640564039457584007913129639935", "115792089237316195423570985008687907853269984665640564039457584007913129639935", "1", "115792089237316195423570985008687907853269984665640564039457584007913129639935"},
		{"115792089237316195423570985008687907853269984665640564039457584007913129639935", "1", "115792089237316195423570985008687907853269984665640564039457584007913129639935", "1"},
		{"16157387885063800092468972531095442600227637936690303362357377535130907802014", "925123444424254823561077285034", "719941466287131323558231662929", "20762213364672992518340486605657941254909946413325109697450958322504414449584"},
		{"69709006495262083753438964270882567809667203355268795714903518762464260067738", "887263329113174494581759872948", "518543687442706248304305296266", "115792089237316195423570985008687907853269984665640564039457584007913129639935"},
		{"87863889970162580245208440663227548203746960243348472696205216504220216118241", "530681276762416477923267193790", "250002842794692296707986335863", "115792089237316195423570985008687907853269984665640564039457584007913129639935"},
		{"41566972108330407139236895720128556017787341847205356916185490192885258906212", "364744754089824788739823854612", "612412446212605494291410711936", "24756738883533665943047591566117958105715971765442627512202339635127795716478"},
		{"33947725049433467350045554180148557448142904607798194645610524279359899319942", "283427037600167819109799088834", "795912206930140882904914252273", "12088900082506786043968624698934530883444698755494298207653557943746057658487"},
	}
	for _, tt := range tests {
		t.Run(tt.maximum+"/"+tt.divisor+"/"+tt.multiplier, func(t *testing.T) {
			maximum := uint256.MustFromDecimal(tt.maximum)
			divisor := uint256.MustFromDecimal(tt.divisor)
			multiplier := uint256.MustFromDecimal(tt.multiplier)
			want := uint256.MustFromDecimal(tt.want)

			got := maxFlooredInput(maximum, divisor, multiplier)
			if got.Cmp(want) != 0 {
				t.Errorf("maxFlooredInput(%s,%s,%s) = %s, want %s", tt.maximum, tt.divisor, tt.multiplier, got.Dec(), want.Dec())
			}
		})
	}
}
