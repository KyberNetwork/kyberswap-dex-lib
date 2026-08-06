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
	testKAI  = "0x0f26bbb8962d73bc891327f14db5162d5279899f" // stable, 18 dec
	testWBTC = "0x0913da6da4b42f538b445599b46bb4622342cf52" // volatile, 8 dec
)

// fixture is a pinned-block capture from live Katana (block 39284109): the vault
// snapshot plus the on-chain quote oracle's outputs (ArbExecutor quoteLeverageAt /
// quoteDeleverageAt / rebalanceState / quote and the raw CollRebalancerMath lib) at the
// same block — the wei-exact parity targets.
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
	RebalanceState     struct {
		Dir     int    `json:"dir"`
		FullIn  string `json:"fullIn"`
		FullOut string `json:"fullOut"`
	} `json:"rebalanceState"`
	QuoteLot struct {
		Dir             int    `json:"dir"`
		StableIn        string `json:"stableIn"`
		VolatileIn      string `json:"volatileIn"`
		StableOut       string `json:"stableOut"`
		VolatileOut     string `json:"volatileOut"`
		CollVaultShares string `json:"collVaultShares"`
	} `json:"quoteLot"`
	LeverageOracle []struct {
		Shares       string `json:"shares"`
		NetStableOut string `json:"netStableOut"`
	} `json:"leverageOracle"`
	DeleverageOracle []struct {
		StableDebtIn string `json:"stableDebtIn"`
		SharesOut    string `json:"sharesOut"`
		StableOut    string `json:"stableOut"`
		VolatileOut  string `json:"volatileOut"`
	} `json:"deleverageOracle"`
	MathlibDirect struct {
		Leverage struct {
			Shares        string `json:"shares"`
			GrossOut      string `json:"grossOut"`
			NewCollateral string `json:"newCollateral"`
			NewDebt       string `json:"newDebt"`
		} `json:"leverage"`
		Deleverage struct {
			StableIn      string `json:"stableIn"`
			SharesOut     string `json:"sharesOut"`
			NewCollateral string `json:"newCollateral"`
			NewDebt       string `json:"newDebt"`
		} `json:"deleverage"`
	} `json:"mathlibDirect"`
}

func bi(t *testing.T, s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	require.True(t, ok, "bad big int fixture %q", s)
	return v
}

func loadFixture(t *testing.T) fixture {
	raw, err := os.ReadFile("testdata/katana_block_39284109.json")
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
		Rebalancer:       "0x4eee1c828b6cafb8cc7bcf44d05f83483e499b23",
		Swapper:          "0x985a6b410f7abe294e4b0fa938d3e8d2f83e79d1",
		CollVault:        "0x3f7da0ade05242d389a86abaf1ba2a85e0563a86",
		ALM:              "0x574c65fda9065288556bde1eccf40afd32244330",
		CvDecimalsOffset: f.CvDecimalsOffset,
		CurveParams:      katanaCurveParams(),
	})
	require.NoError(t, err)

	sim, err := NewPoolSimulator(entity.Pool{
		Address:  "0x985a6b410f7abe294e4b0fa938d3e8d2f83e79d1",
		Exchange: "everlong-collvault",
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: testKAI, Decimals: 18, Swappable: true},
			{Address: testWBTC, Decimals: 8, Swappable: true},
		},
		Reserves:    entity.PoolReserves{f.AlmStableReserve, f.AlmVolatileReserve},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	})
	require.NoError(t, err)
	return sim
}

// TestLeverageQuoteMatchesOnChainOracle: quoteLeverageAt must be wei-exact vs the
// on-chain oracle sweep captured at the pinned block.
func TestLeverageQuoteMatchesOnChainOracle(t *testing.T) {
	f := loadFixture(t)
	state := vaultStateFromFixture(t, f)
	cp := katanaCurveParams()

	for _, tc := range f.LeverageOracle {
		net, ok := cp.quoteLeverageAt(state, bi(t, tc.Shares))
		require.True(t, ok)
		require.Zero(t, bi(t, tc.NetStableOut).Cmp(net),
			"shares=%s got=%s want=%s", tc.Shares, net, tc.NetStableOut)
	}
}

// TestDeleverageQuoteMatchesOnChainOracle: shares out and both freed legs must be
// wei-exact vs the on-chain quoteDeleverageAt sweep.
func TestDeleverageQuoteMatchesOnChainOracle(t *testing.T) {
	f := loadFixture(t)
	state := vaultStateFromFixture(t, f)
	cp := katanaCurveParams()

	for _, tc := range f.DeleverageOracle {
		sharesOut, _, _, recovery := cp.deleverageQuote(state.Collateral, state.Debt,
			state.PriceWad, cp.LeverageRatioWad, state.SpreadPpm, bi(t, tc.StableDebtIn))
		require.False(t, recovery)
		require.Zero(t, bi(t, tc.SharesOut).Cmp(sharesOut),
			"stableDebtIn=%s sharesOut got=%s want=%s", tc.StableDebtIn, sharesOut, tc.SharesOut)

		stableOut, volatileOut, ok := state.previewTokenAmounts(sharesOut, false)
		require.True(t, ok)
		require.Zero(t, bi(t, tc.StableOut).Cmp(stableOut),
			"stableDebtIn=%s stableOut got=%s want=%s", tc.StableDebtIn, stableOut, tc.StableOut)
		require.Zero(t, bi(t, tc.VolatileOut).Cmp(volatileOut),
			"stableDebtIn=%s volatileOut got=%s want=%s", tc.StableDebtIn, volatileOut, tc.VolatileOut)
	}
}

// TestRawMathMatchesDeployedMathLib: the raw CR-math quotes (gross out + post state)
// must match the deployed CollRebalancerMath library called directly.
func TestRawMathMatchesDeployedMathLib(t *testing.T) {
	f := loadFixture(t)
	state := vaultStateFromFixture(t, f)
	cp := katanaCurveParams()

	lev := f.MathlibDirect.Leverage
	grossOut, newColl, newDebt := cp.leverageQuote(state.Collateral, state.Debt,
		state.PriceWad, cp.LeverageRatioWad, state.SpreadPpm, bi(t, lev.Shares))
	require.Zero(t, bi(t, lev.GrossOut).Cmp(grossOut))
	require.Zero(t, bi(t, lev.NewCollateral).Cmp(newColl))
	require.Zero(t, bi(t, lev.NewDebt).Cmp(newDebt))

	dlv := f.MathlibDirect.Deleverage
	sharesOut, newColl, newDebt, recovery := cp.deleverageQuote(state.Collateral, state.Debt,
		state.PriceWad, cp.LeverageRatioWad, state.SpreadPpm, bi(t, dlv.StableIn))
	require.False(t, recovery)
	require.Zero(t, bi(t, dlv.SharesOut).Cmp(sharesOut))
	require.Zero(t, bi(t, dlv.NewCollateral).Cmp(newColl))
	require.Zero(t, bi(t, dlv.NewDebt).Cmp(newDebt))
}

// TestRebalanceStateMatchesOnChainLens: direction + MAX-lot sizing (the physical-CR
// floor bisection) must be wei-exact vs the executor's rebalanceState()/quote() views.
func TestRebalanceStateMatchesOnChainLens(t *testing.T) {
	f := loadFixture(t)
	state := vaultStateFromFixture(t, f)
	cp := katanaCurveParams()

	dir, fullIn, fullOut, ok := cp.rebalanceState(state)
	require.True(t, ok)
	require.Equal(t, f.RebalanceState.Dir, dir)
	require.Zero(t, bi(t, f.RebalanceState.FullIn).Cmp(fullIn))
	require.Zero(t, bi(t, f.RebalanceState.FullOut).Cmp(fullOut))

	// the lot's token legs (quote() view) come from previewTokenAmounts at the max lot
	stableIn, volatileIn, pok := state.previewTokenAmounts(fullIn, true)
	require.True(t, pok)
	require.Zero(t, bi(t, f.QuoteLot.StableIn).Cmp(stableIn))
	require.Zero(t, bi(t, f.QuoteLot.VolatileIn).Cmp(volatileIn))
}

// TestCalcAmountOutLeverage: WBTC -> KAI end-to-end. The quoted net must equal
// quoteLeverageAt at the sized shares, the volatile leg must fit within the input, and
// the unconsumed input must be surfaced.
func TestCalcAmountOutLeverage(t *testing.T) {
	sim := newTestPoolSimulator(t)
	f := loadFixture(t)
	amountIn := bi(t, f.QuoteLot.VolatileIn) // the full lot's volatile leg

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testWBTC, Amount: amountIn},
		TokenOut:      testKAI,
	})
	require.NoError(t, err)
	require.True(t, result.TokenAmountOut.Amount.Sign() > 0)

	si, ok := result.SwapInfo.(SwapInfo)
	require.True(t, ok)
	require.True(t, si.IsLeverage)
	require.True(t, si.VolatileLeg.Cmp(amountIn) <= 0, "must not consume more than the input")

	spent := new(big.Int).Add(si.VolatileLeg, result.RemainingTokenAmountIn.Amount)
	require.Zero(t, amountIn.Cmp(spent), "consumed + remaining must equal the input")

	// the quote must be exactly the composed oracle value at the sized shares
	state := vaultStateFromFixture(t, f)
	cp := katanaCurveParams()
	oracleNet, ok2 := cp.quoteLeverageAt(state, si.CollVaultShares)
	require.True(t, ok2)
	require.Zero(t, oracleNet.Cmp(result.TokenAmountOut.Amount))
}

// TestCalcAmountOutDeleverage: KAI -> WBTC end-to-end. The forward net must fit the
// input and the output must be exactly the freed volatile leg at the sized gross.
func TestCalcAmountOutDeleverage(t *testing.T) {
	sim := newTestPoolSimulator(t)
	amountIn, _ := new(big.Int).SetString("500000000000000000000", 10) // 500 KAI net budget

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testKAI, Amount: amountIn},
		TokenOut:      testWBTC,
	})
	require.NoError(t, err)
	require.True(t, result.TokenAmountOut.Amount.Sign() > 0)

	si, ok := result.SwapInfo.(SwapInfo)
	require.True(t, ok)
	require.False(t, si.IsLeverage)

	net := new(big.Int).Sub(si.GrossStableIn, si.StableLeg)
	require.True(t, net.Cmp(amountIn) <= 0, "forward net must fit within the input")
	spent := new(big.Int).Add(net, result.RemainingTokenAmountIn.Amount)
	require.Zero(t, amountIn.Cmp(spent), "net + remaining must equal the input")

	// re-derive the legs at the chosen gross — must agree with the quote
	f := loadFixture(t)
	state := vaultStateFromFixture(t, f)
	cp := katanaCurveParams()
	stableOut, volatileOut, lok := cp.deleverageLegsAt(state, si.GrossStableIn)
	require.True(t, lok)
	require.Zero(t, stableOut.Cmp(si.StableLeg))
	require.Zero(t, volatileOut.Cmp(result.TokenAmountOut.Amount))
}

// TestCalcAmountOutIsPureAndDeterministic: repeated quoting must not drift.
func TestCalcAmountOutIsPureAndDeterministic(t *testing.T) {
	sim := newTestPoolSimulator(t)
	amountIn, _ := new(big.Int).SetString("1000000000000000000000", 10)
	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testKAI, Amount: amountIn},
		TokenOut:      testWBTC,
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
	amountIn, _ := new(big.Int).SetString("1000000000000000000000", 10)
	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testKAI, Amount: amountIn},
		TokenOut:      testWBTC,
	}
	first, err := sim.CalcAmountOut(params)
	require.NoError(t, err)

	backup := sim.CloneState()

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testKAI, Amount: amountIn},
		TokenAmountOut: *first.TokenAmountOut,
		Fee:            *first.Fee,
		SwapInfo:       first.SwapInfo,
	})

	second, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	require.NotZero(t, second.TokenAmountOut.Amount.Cmp(first.TokenAmountOut.Amount),
		"a deleverage fill moves the position — the next quote must differ")

	fromBackup, err := backup.CalcAmountOut(params)
	require.NoError(t, err)
	require.Zero(t, first.TokenAmountOut.Amount.Cmp(fromBackup.TokenAmountOut.Amount),
		"CloneState must fully insulate the backup from UpdateBalance")
}
