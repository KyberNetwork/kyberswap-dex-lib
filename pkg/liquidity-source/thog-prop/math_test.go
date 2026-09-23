package thogprop

import (
	"math/big"
	"testing"
)

// stateBlock107066625 is the live makerSnapshot() state at Monad block
// 107066625, captured while integrating this package. Used to verify
// ExactQuote against four independently-called makerQuoteExactInput()
// results at that exact block -- see the fixtures below.
func stateBlock107066625() *State {
	bi := func(s string) *big.Int {
		v, ok := new(big.Int).SetString(s, 10)
		if !ok {
			panic("bad literal: " + s)
		}
		return v
	}
	return &State{
		V1:             bi("2994239055849461445655022230385668861908425422835416817831970788550610092030"),
		V2:             bi("1124588846958885914322232869647557697816726"),
		V3:             bi("853655712402932"),
		R1:             bi("2014949657059124526062761958401416449565747983888532651381762"),
		R2:             bi("294984522595088000014157334"),
		R3:             bi("43066045109633176"),
		I1:             bi("26959946667149500150766263744785391798464467030925802040175863822145"),
		I2:             bi("112083"),
		P1:             bi("57896044618658097711785492504343953926634992332820282019728792003956564819968"),
		P2:             bi("57896044618658097711785492504343953926634992332820282019728792003956564819968"),
		GloballyPaused: false,
		RiskV3Ready:    true,
		PairRiskReady:  true,
		Balances: []*big.Int{
			bi("36406748694"),              // USDC
			bi("20581810"),                 // AUSD
			bi("2227929976"),               // USDT0
			bi("171154752028809148389158"), // WMON
			bi("2698350120294236217"),      // WETH
			bi("895623"),                   // WBTC
			bi("7523178"),                  // cbBTC
			bi("1846142"),                  // XAUt0
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
			amountIn, ok := new(big.Int).SetString(tt.amountIn, 10)
			if !ok {
				t.Fatalf("bad amountIn literal %q", tt.amountIn)
			}
			wantOut, ok := new(big.Int).SetString(tt.wantOut, 10)
			if !ok {
				t.Fatalf("bad wantOut literal %q", tt.wantOut)
			}

			gotOut, postedBlock, err := ExactQuote(s, tok(tt.tokenInIdx), tok(tt.tokenOutIdx), amountIn, executionBlock, false)
			if err != nil {
				t.Fatalf("ExactQuote() error = %v", err)
			}
			if gotOut.Cmp(wantOut) != 0 {
				t.Errorf("amountOut = %s, want %s (diff %s)", gotOut, wantOut, new(big.Int).Sub(gotOut, wantOut))
			}
			if postedBlock != executionBlock {
				t.Errorf("postedBlock = %d, want %d", postedBlock, executionBlock)
			}
		})
	}
}

func TestExactQuote_FastLaneHaircut(t *testing.T) {
	s := stateBlock107066625()
	amountIn := big.NewInt(1000000000)

	base, _, err := ExactQuote(s, tok(0), tok(3), amountIn, 107066625, false)
	if err != nil {
		t.Fatalf("ExactQuote(fastLaneHot=false) error = %v", err)
	}
	hot, _, err := ExactQuote(s, tok(0), tok(3), amountIn, 107066625, true)
	if err != nil {
		t.Fatalf("ExactQuote(fastLaneHot=true) error = %v", err)
	}

	want := new(big.Int).Div(new(big.Int).Mul(base, big.NewInt(9990)), big.NewInt(10000))
	if hot.Cmp(want) != 0 {
		t.Errorf("fastLaneHot amountOut = %s, want %s (9990/10000 of %s)", hot, want, base)
	}
	if hot.Cmp(base) >= 0 {
		t.Errorf("fastLaneHot amountOut %s should be strictly less than cold amountOut %s", hot, base)
	}
}

func TestExactQuote_SameTokenRejected(t *testing.T) {
	s := stateBlock107066625()
	if _, _, err := ExactQuote(s, tok(0), tok(0), big.NewInt(1000000), 107066625, false); err == nil {
		t.Error("expected an error quoting tokenIn == tokenOut, got nil")
	}
}

func TestExactQuote_MaxAgeExceededReturnsSentinel(t *testing.T) {
	s := stateBlock107066625()
	maxAge := bits(s.R1, 198, 8).Uint64()
	_, _, err := ExactQuote(s, tok(0), tok(3), big.NewInt(1000000000), 107066625+maxAge+1, false)
	if err != ErrStalePrice {
		t.Errorf("got err = %v, want ErrStalePrice", err)
	}
}
