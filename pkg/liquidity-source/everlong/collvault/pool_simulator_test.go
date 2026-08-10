package everlongcollvault

import (
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	testNECT = "0x1ce0a25d13ce4d52071ae7e02cf1f6606f4c79d3" // stable, 18 dec
	testWBTC = "0x0555e30da8f98308edb960aa94c0db47230d2b9c" // volatile, 8 dec
)

// fixture is a pinned-block state capture from live Berachain (block 24710262): the
// rebalancer + ALM adapter + CollVault snapshot the simulator prices from. Quote-level
// wei-exactness is asserted against the shipped-library fixture grid in math_test.go;
// the tests here exercise the composed venue behavior on a real state.
type fixture struct {
	Block              int64  `json:"block"`
	C                  string `json:"C"`
	D                  string `json:"D"`
	R                  string `json:"R"`
	Spread             int64  `json:"spread"`
	AlmStableReserve   string `json:"almStableReserve"`
	AlmVolatileReserve string `json:"almVolatileReserve"`
	AlmSupply          string `json:"almSupply"`
	CvTotalAssets      string `json:"cvTotalAssets"`
	CvTotalSupply      string `json:"cvTotalSupply"`
	CvDecimalsOffset   uint8  `json:"cvDecimalsOffset"`
	WithdrawFeeBp      int64  `json:"withdrawFeeBp"`
	RefStableReserve   string `json:"refStableReserve"`
	RefAssetReserve    string `json:"refAssetReserve"`
	RefRawReferenceWad string `json:"refRawReferenceWad"`
}

func bi(t *testing.T, s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	require.True(t, ok, "bad big int fixture %q", s)
	return v
}

func loadFixture(t *testing.T) fixture {
	raw, err := os.ReadFile("testdata/berachain_block_24710262.json")
	require.NoError(t, err)
	var f fixture
	require.NoError(t, json.Unmarshal(raw, &f))
	return f
}

func vaultStateFromFixture(t *testing.T, f fixture) *VaultState {
	return &VaultState{
		Collateral: bi(t, f.C), Debt: bi(t, f.D), PriceWad: bi(t, f.R),
		SpreadPpm:        big.NewInt(f.Spread),
		AlmStableReserve: bi(t, f.AlmStableReserve), AlmVolatileReserve: bi(t, f.AlmVolatileReserve),
		AlmSupply:     bi(t, f.AlmSupply),
		CvTotalAssets: bi(t, f.CvTotalAssets), CvTotalSupply: bi(t, f.CvTotalSupply),
		CvDecimalsOffset: f.CvDecimalsOffset, WithdrawFeeBp: big.NewInt(f.WithdrawFeeBp),
		RefStableReserve: bi(t, f.RefStableReserve), RefAssetReserve: bi(t, f.RefAssetReserve),
		RefRawReferenceWad: bi(t, f.RefRawReferenceWad),
	}
}

func newTestPoolSimulator(t *testing.T) *PoolSimulator {
	f := loadFixture(t)
	state := vaultStateFromFixture(t, f)
	extraBytes, err := json.Marshal(state)
	require.NoError(t, err)
	staticExtraBytes, err := json.Marshal(StaticExtra{
		Rebalancer:       "0xa6b848d899189d263a9398f1df4534af7b06d6b3",
		Swapper:          "0x27775ec38e2b394738b73c0d25f63e20063df054",
		CollVault:        "0x9e7f375c351a251e80eb89ad33ca62b270fd9b4a",
		ALM:              "0xbd10884d6b55eda1d872cd5108b8aabdc0c3f6ca",
		CvDecimalsOffset: f.CvDecimalsOffset,
		CurveParams:      berachainCurveParams(),
	})
	require.NoError(t, err)

	sim, err := NewPoolSimulator(entity.Pool{
		Address:  "0x27775ec38e2b394738b73c0d25f63e20063df054",
		Exchange: "everlong-collvault",
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: testNECT, Decimals: 18, Swappable: true},
			{Address: testWBTC, Decimals: 8, Swappable: true},
		},
		Reserves:    entity.PoolReserves{f.AlmStableReserve, f.AlmVolatileReserve},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	})
	require.NoError(t, err)
	return sim
}

// quoteableDirection picks the direction the pinned state actually accepts: the CR
// curve realigns one way at a time, so exactly one of the two should quote.
func quoteableDirection(t *testing.T, sim *PoolSimulator) (tokenIn, tokenOut string, amountIn *big.Int) {
	f := loadFixture(t)
	state := vaultStateFromFixture(t, f)
	cp := berachainCurveParams()
	dir, fullIn, fullOut := cp.rebalanceState(state)
	require.NotZero(t, dir, "pinned state must accept a fill in one direction")
	require.True(t, fullIn.Sign() > 0 && fullOut.Sign() > 0)
	if dir == 1 { // leverage: volatile in
		_, volatileIn, ok := state.previewTokenAmounts(fullIn, true)
		require.True(t, ok)
		return testWBTC, testNECT, volatileIn
	}
	// deleverage: net stable in
	stableOut, _, ok := cp.deleverageLegsAt(state, fullIn)
	require.True(t, ok)
	return testNECT, testWBTC, new(big.Int).Sub(fullIn, stableOut)
}

// TestCalcAmountOutComposesQuotes: end-to-end on the pinned state. The consumed leg
// plus the reported remainder must equal the input, and the output must re-derive
// exactly from the checked quotes at the sizes SwapInfo reports.
func TestCalcAmountOutComposesQuotes(t *testing.T) {
	sim := newTestPoolSimulator(t)
	tokenIn, tokenOut, amountIn := quoteableDirection(t, sim)

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
		TokenOut:      tokenOut,
	})
	require.NoError(t, err)
	require.True(t, result.TokenAmountOut.Amount.Sign() > 0)

	si, ok := result.SwapInfo.(SwapInfo)
	require.True(t, ok)

	f := loadFixture(t)
	state := vaultStateFromFixture(t, f)
	cp := berachainCurveParams()
	if si.IsLeverage {
		require.True(t, si.VolatileLeg.Cmp(amountIn) <= 0, "must not consume more than the input")
		spent := new(big.Int).Add(si.VolatileLeg, result.RemainingTokenAmountIn.Amount)
		require.Zero(t, amountIn.Cmp(spent), "consumed + remaining must equal the input")
		oracleNet, ok2 := cp.quoteLeverageAt(state, si.CollVaultShares)
		require.True(t, ok2)
		require.Zero(t, oracleNet.Cmp(result.TokenAmountOut.Amount))
	} else {
		net := new(big.Int).Sub(si.GrossStableIn, si.StableLeg)
		require.True(t, net.Cmp(amountIn) <= 0, "forward net must fit within the input")
		spent := new(big.Int).Add(net, result.RemainingTokenAmountIn.Amount)
		require.Zero(t, amountIn.Cmp(spent), "net + remaining must equal the input")
		stableOut, volatileOut, lok := cp.deleverageLegsAt(state, si.GrossStableIn)
		require.True(t, lok)
		require.Zero(t, stableOut.Cmp(si.StableLeg))
		require.Zero(t, volatileOut.Cmp(result.TokenAmountOut.Amount))
	}
}

// TestCalcAmountOutIsPureAndDeterministic: repeated quoting must not drift.
func TestCalcAmountOutIsPureAndDeterministic(t *testing.T) {
	sim := newTestPoolSimulator(t)
	tokenIn, tokenOut, amountIn := quoteableDirection(t, sim)
	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
		TokenOut:      tokenOut,
	}
	first, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	for i := 0; i < 3; i++ {
		again, err := sim.CalcAmountOut(params)
		require.NoError(t, err)
		require.Zero(t, first.TokenAmountOut.Amount.Cmp(again.TokenAmountOut.Amount))
	}
}

// TestUpdateBalanceAndClone: after a fill the same request must price differently
// (state moved), while a pre-update clone still prices the original.
func TestUpdateBalanceAndClone(t *testing.T) {
	sim := newTestPoolSimulator(t)
	tokenIn, tokenOut, amountIn := quoteableDirection(t, sim)
	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
		TokenOut:      tokenOut,
	}
	first, err := sim.CalcAmountOut(params)
	require.NoError(t, err)

	backup := sim.CloneState()

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: tokenIn, Amount: amountIn},
		TokenAmountOut: *first.TokenAmountOut,
		Fee:            *first.Fee,
		SwapInfo:       first.SwapInfo,
	})

	second, err := sim.CalcAmountOut(params)
	if err == nil {
		require.NotZero(t, second.TokenAmountOut.Amount.Cmp(first.TokenAmountOut.Amount),
			"a fill moves the position — the next quote must differ")
	}

	fromBackup, err := backup.CalcAmountOut(params)
	require.NoError(t, err)
	require.Zero(t, first.TokenAmountOut.Amount.Cmp(fromBackup.TokenAmountOut.Amount),
		"CloneState must fully insulate the backup from UpdateBalance")
}
