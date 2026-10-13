package limitorder

import (
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

// refOrder is an order as the executor sees it on-chain.
type refOrder struct {
	making, taking     *big.Int // order totals
	remaining          *big.Int // on-chain remaining making
	balance, allowance *big.Int // maker's, for the LO contract
	feePct             int64    // in FEE_UNITS (100_000) = 10 * makerTokenFeePercent
	takerAssetFee      bool
}

const refFeeUnits = 100_000

func ceilDiv(a, b *big.Int) *big.Int {
	q, r := new(big.Int).QuoRem(a, b, new(big.Int))
	if r.Sign() > 0 {
		q.Add(q, big.NewInt(1))
	}
	return q
}

func bigMin(a, b *big.Int) *big.Int {
	if a.Cmp(b) < 0 {
		return new(big.Int).Set(a)
	}
	return new(big.Int).Set(b)
}

// refExecutorFill ports the on-chain LO executor's fill loop plus the amount and fee math of the LO
// contract's fillOrderTo, for orders of distinct makers with a non-zero fee recipient.
var refUsedDust bool // set when the executor fills an order with what a taker-fee partial fill left

func refExecutorFill(orders []refOrder, amountIn *big.Int) (amountOut, unused *big.Int) {
	refUsedDust = false
	afterTakerPartial := false
	amountIn = new(big.Int).Set(amountIn)
	amountOut = new(big.Int)
	fu := big.NewInt(refFeeUnits)
	for _, o := range orders {
		// executor
		making := bigMin(bigMin(o.remaining, o.balance), o.allowance)
		taking := new(big.Int).Quo(new(big.Int).Mul(o.taking, making), o.making)
		pct := big.NewInt(o.feePct)
		takerFee := o.feePct != 0 && o.takerAssetFee
		partial := false
		if takerFee {
			fee := ceilDiv(new(big.Int).Mul(pct, taking), fu)
			if new(big.Int).Add(taking, fee).Cmp(amountIn) > 0 {
				taking = new(big.Int).Quo(new(big.Int).Mul(amountIn, fu), new(big.Int).Add(fu, pct))
				partial = true
			}
		} else {
			taking = bigMin(taking, amountIn)
		}
		if taking.Sign() == 0 || new(big.Int).Mul(o.making, taking).Cmp(o.taking) < 0 {
			continue
		}
		// the LO contract's fillOrderTo with AmountData{takingAmount: taking}
		cMaking := new(big.Int).Quo(new(big.Int).Mul(taking, o.making), o.taking) // getMakerAmount
		if cMaking.Cmp(o.remaining) > 0 {
			cMaking.Set(o.remaining)
			taking = ceilDiv(new(big.Int).Mul(cMaking, o.taking), o.making) // getTakerAmount
		}
		if cMaking.Sign() == 0 || cMaking.Cmp(o.balance) > 0 || cMaking.Cmp(o.allowance) > 0 {
			continue // transfer from the maker reverts; the executor catches it
		}
		var fee, actualTaking *big.Int
		if o.feePct != 0 && o.takerAssetFee {
			fee = ceilDiv(new(big.Int).Mul(taking, pct), fu)
			actualTaking = new(big.Int).Add(taking, fee)
		} else if o.feePct != 0 {
			fee = ceilDiv(new(big.Int).Mul(cMaking, pct), fu)
			actualTaking = taking
		} else {
			fee, actualTaking = new(big.Int), taking
		}
		if actualTaking.Cmp(amountIn) > 0 {
			continue // the executor can't pay; the transfer reverts
		}
		amountIn.Sub(amountIn, actualTaking)
		if afterTakerPartial {
			refUsedDust = true
		}
		afterTakerPartial = partial
		if takerFee {
			amountOut.Add(amountOut, cMaking)
		} else {
			amountOut.Add(amountOut, new(big.Int).Sub(cMaking, fee))
		}
		if amountIn.Sign() == 0 {
			break
		}
	}
	return amountOut, amountIn
}

func refRandAmount(rng *rand.Rand) *big.Int {
	base := big.NewInt(1 + rng.Int64N(1_000_000))
	return base.Mul(base, new(big.Int).Exp(big.NewInt(10), big.NewInt([]int64{0, 0, 2, 6, 18}[rng.IntN(5)]), nil))
}

func refFrac(rng *rand.Rand, x *big.Int) *big.Int {
	return new(big.Int).Quo(new(big.Int).Mul(x, big.NewInt(rng.Int64N(1001))), big.NewInt(1000))
}

// The simulator must quote exactly what the executor delivers on random books, except that the
// executor also spends the few wei a taker-fee partial fill leaves on later orders, which the
// simulator drops; it may never quote more.
func TestCalcAmountOut_MatchesExecutor(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 0))
	var same, simLower int
	maxLower := new(big.Int)
	for c := range 20000 {
		n := 1 + rng.IntN(6)
		refs := make([]refOrder, n)
		orders := make([]*order, n)
		for i := range refs {
			making, taking := refRandAmount(rng), refRandAmount(rng)
			remaining := making
			if rng.IntN(2) == 0 {
				remaining = refFrac(rng, making)
			}
			balance := new(big.Int).Set(making)
			if rng.IntN(3) == 0 {
				balance = refFrac(rng, making)
			}
			r := refOrder{making: making, taking: taking, remaining: remaining, balance: balance, allowance: balance,
				takerAssetFee: rng.IntN(2) == 0}
			p := []int64{0, 5, 30, 100}[rng.IntN(4)]
			r.feePct = p * 10
			refs[i] = r
			filledMaking := new(big.Int).Sub(making, remaining)
			orders[i] = &order{
				ID: int64(i + 1), Maker: fmt.Sprintf("0xmaker%d", i), MakerAsset: "m", TakerAsset: "t",
				MakingAmount: making, TakingAmount: taking,
				FilledMakingAmount:    filledMaking,
				FilledTakingAmount:    new(big.Int).Quo(new(big.Int).Mul(filledMaking, taking), making),
				AvailableMakingAmount: bigMin(remaining, balance),
				MakerBalanceAllowance: balance,
				MakerTokenFeePercent:  uint32(p), IsTakerAssetFee: r.takerAssetFee,
			}
		}
		extra, _ := json.Marshal(Extra{BuyOrders: orders})
		sim, err := NewPoolSimulator(entity.Pool{Tokens: []*entity.PoolToken{{Address: "a"}, {Address: "b"}},
			Reserves: entity.PoolReserves{"0", "0"}, StaticExtra: `{"ContractAddress":""}`, Extra: string(extra)})
		require.NoError(t, err)
		amountIn := refRandAmount(rng)
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: "a", Amount: amountIn},
			TokenOut: "b", Limit: swaplimit.NewInventory("", sim.CalculateLimit())})
		if err != nil {
			continue // the simulator only quotes when it can place all of amountIn
		}
		want, _ := refExecutorFill(refs, amountIn)
		switch d := new(big.Int).Sub(want, res.TokenAmountOut.Amount); d.Sign() {
		case 0:
			same++
		case 1:
			require.True(t, refUsedDust, "case %d: simulator %s below executor %s without taker-fee dust",
				c, res.TokenAmountOut.Amount, want)
			simLower++
			if d.Cmp(maxLower) > 0 {
				maxLower = d
			}
		default:
			require.Failf(t, "simulator quotes more than the executor delivers",
				"case %d: simulator %s, executor %s, amountIn %s", c, res.TokenAmountOut.Amount, want, amountIn)
		}
	}
	t.Logf("simulator == executor: %d, below by taker-fee dust: %d (max %s wei)", same, simLower, maxLower)
	require.Positive(t, same)
}
