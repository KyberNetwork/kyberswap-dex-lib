package altfun

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	bouncetech "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bounce-tech"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type parityBT struct {
	ExchangeRate   string `json:"exchangeRate"`
	RedemptionFee  string `json:"redemptionFee"`
	TargetLeverage string `json:"targetLeverage"`
	MinTxSize      string `json:"minTxSize"`
	BaseBalance    string `json:"baseBalance"`
	MintPaused     bool   `json:"mintPaused"`
}

type parityPair struct {
	ReserveToken           string `json:"reserveToken"`
	ReserveAsset           string `json:"reserveAsset"`
	K                      string `json:"k"`
	TokenBalance           string `json:"tokenBalance"`
	GraduationThresholdUsd string `json:"graduationThresholdUsd"`
	Lifecycle              uint8  `json:"lifecycle"`
	BuyFeeBps              uint64 `json:"buyFeeBps"`
	SellFeeBps             uint64 `json:"sellFeeBps"`
}

type parityStep struct {
	TokenIn       string `json:"tokenIn"`
	AmountIn      string `json:"amountIn"`
	Err           string `json:"err"`
	AmountOut     string `json:"amountOut"`
	Fee           string `json:"fee"`
	Remaining     string `json:"remaining"`
	Gas           int64  `json:"gas"`
	IsSell        bool   `json:"isSell"`
	AmountInUsed  string `json:"amountInUsed"`
	BaseToConvert string `json:"baseToConvert"`
	ReserveToken  string `json:"reserveToken"`
	ReserveAsset  string `json:"reserveAsset"`
	TokenBalance  string `json:"tokenBalance"`
	Lifecycle     uint8  `json:"lifecycle"`
}

type parityScenario struct {
	Name  string       `json:"name"`
	BT    parityBT     `json:"bt"`
	Pair  parityPair   `json:"pair"`
	Steps []parityStep `json:"steps"`
}

func decOrEmpty(v *uint256.Int) string {
	if v == nil {
		return ""
	}
	return v.Dec()
}

func buildParitySim(t *testing.T, sc parityScenario) *PoolSimulator {
	t.Helper()
	btBytes, err := json.Marshal(bouncetech.Extra{
		ExchangeRate:       uint256.MustFromDecimal(sc.BT.ExchangeRate),
		RedemptionFee:      uint256.MustFromDecimal(sc.BT.RedemptionFee),
		TargetLeverage:     uint256.MustFromDecimal(sc.BT.TargetLeverage),
		MinTransactionSize: uint256.MustFromDecimal(sc.BT.MinTxSize),
		MintPaused:         sc.BT.MintPaused,
	})
	require.NoError(t, err)
	btSim, err := bouncetech.NewPoolSimulator(entity.Pool{
		Address:  testLT,
		Exchange: bouncetech.DexType,
		Type:     bouncetech.DexType,
		Tokens: []*entity.PoolToken{
			{Address: testUSDC, Swappable: true},
			{Address: testLT, Swappable: true},
		},
		Reserves:    entity.PoolReserves{sc.BT.BaseBalance, "1000000000000000000000000000"},
		Extra:       string(btBytes),
		StaticExtra: `{"usdc":"` + testUSDC + `"}`,
	})
	require.NoError(t, err)

	extraBytes, err := json.Marshal(Extra{
		ReserveToken: uint256.MustFromDecimal(sc.Pair.ReserveToken),
		ReserveAsset: uint256.MustFromDecimal(sc.Pair.ReserveAsset),
		K:            uint256.MustFromDecimal(sc.Pair.K),
		TokenBalance: uint256.MustFromDecimal(sc.Pair.TokenBalance),
		Lifecycle:    Lifecycle(sc.Pair.Lifecycle),
	})
	require.NoError(t, err)
	staticBytes, err := json.Marshal(StaticExtra{
		LTAddress:              testLT,
		USDC:                   testUSDC,
		ZapAddress:             testZap,
		BuyFeeBps:              sc.Pair.BuyFeeBps,
		SellFeeBps:             sc.Pair.SellFeeBps,
		BasePool:               testLT,
		GraduationThresholdUsd: uint256.MustFromDecimal(sc.Pair.GraduationThresholdUsd),
	})
	require.NoError(t, err)

	s, err := NewPoolSimulator(entity.Pool{
		Address:  testMeme,
		Exchange: DexType,
		Type:     DexType,
		Reserves: entity.PoolReserves{sc.Pair.ReserveToken, sc.Pair.TokenBalance},
		Tokens: []*entity.PoolToken{
			{Address: testUSDC, Swappable: true},
			{Address: testMeme, Swappable: true},
		},
		Extra:       string(extraBytes),
		StaticExtra: string(staticBytes),
	}, map[string]pool.IPoolSim{testLT: btSim})
	require.NoError(t, err)
	return s
}

// TestParityAgainstLegacyGoldens replays scenarios captured at db2a3162b (last commit where alt-fun built and
// tested). Each step must match the golden on error, amounts, fee, gas, remaining input, SwapInfo and the state
// after UpdateState. Goldens are fixed; a mismatch means the port changed behavior.
func TestParityAgainstLegacyGoldens(t *testing.T) {
	b, err := os.ReadFile("testdata/parity_goldens.json")
	require.NoError(t, err)
	var scenarios []parityScenario
	require.NoError(t, json.Unmarshal(b, &scenarios))
	require.NotEmpty(t, scenarios)

	for _, sc := range scenarios {
		t.Run(sc.Name, func(t *testing.T) {
			s := buildParitySim(t, sc)
			for i, st := range sc.Steps {
				tokenIn, tokenOut := testUSDC, testMeme
				if st.TokenIn == "meme" {
					tokenIn, tokenOut = testMeme, testUSDC
				}
				var amt uint256.Int
				require.NoError(t, amt.SetFromDecimal(st.AmountIn))

				res, err := s.CalcAmtOut(pool.CalcAmtOutParams{
					TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: amt},
					TokenOut:      tokenOut,
				})
				if st.Err != "" {
					require.EqualError(t, err, st.Err, "step %d", i)
					continue
				}
				require.NoError(t, err, "step %d", i)
				require.Equal(t, st.AmountOut, decOrEmpty(&res.TokenAmountOut.Amount), "step %d amountOut", i)
				require.Equal(t, st.Fee, decOrEmpty(&res.Fee.Amount), "step %d fee", i)
				require.Equal(t, st.Gas, res.Gas, "step %d gas", i)

				remaining := ""
				if res.RemainingTokenAmountIn != nil {
					remaining = decOrEmpty(&res.RemainingTokenAmountIn.Amount)
				}
				require.Equal(t, st.Remaining, remaining, "step %d remaining", i)

				si := res.SwapInfo.(SwapInfo)
				require.Equal(t, st.IsSell, si.IsSell, "step %d isSell", i)
				require.Equal(t, st.AmountInUsed, decOrEmpty(si.AmountInUsed), "step %d amountInUsed", i)
				require.Equal(t, st.BaseToConvert, decOrEmpty(si.BaseToConvert), "step %d baseToConvert", i)

				s.UpdateState(pool.UpdateStateParams{
					TokenAmountIn:  pool.TokenAmt{Token: tokenIn, Amount: amt},
					TokenAmountOut: *res.TokenAmountOut,
					Fee:            *res.Fee,
					SwapInfo:       res.SwapInfo,
				})
				require.Equal(t, st.ReserveToken, decOrEmpty(s.reserveToken), "step %d reserveToken", i)
				require.Equal(t, st.ReserveAsset, decOrEmpty(s.reserveAsset), "step %d reserveAsset", i)
				require.Equal(t, st.TokenBalance, decOrEmpty(s.tokenBalance), "step %d tokenBalance", i)
				require.Equal(t, st.Lifecycle, uint8(s.lifecycle), "step %d lifecycle", i)
			}
		})
	}
}
