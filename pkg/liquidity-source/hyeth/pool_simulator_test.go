package hyeth

import (
	"math"
	"math/big"
	"strings"
	"sync"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// getPoolJSON is the hyETH pool entity used by the simulator tests.
const getPoolJSON = "{\"address\":\"0xcb1eea349f25288627f008c5e2a69b684bdddf49\",\"exchange\":\"hyeth\",\"type\":\"hyeth\",\"timestamp\":1745235076,\"reserves\":[\"4946361947932843870115\",\"5005345678839792956730\"],\"tokens\":[{\"address\":\"0xc4506022fb8090774e8a628d5084eed61d9b99ee\",\"name\":\"hyeth\",\"symbol\":\"hyETH\",\"decimals\":18,\"weight\":1,\"swappable\":true},{\"address\":\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\",\"name\":\"WETH\",\"symbol\":\"WETH\",\"decimals\":18,\"weight\":1,\"swappable\":true}],\"extra\":\"{\\\"feeI\\\":\\\"0\\\",\\\"feeR\\\":\\\"0\\\",\\\"comp\\\":\\\"0x701907283a57ff77e255c3f1aad790466b8ce4ef\\\",\\\"compSup\\\":\\\"4946361947932843870115\\\",\\\"compAss\\\":\\\"5005345678839792956730\\\",\\\"compHyb\\\":\\\"1015907674038080762600\\\",\\\"hySup\\\":\\\"809233550815085194542\\\",\\\"dpru\\\":\\\"1255394901774434537\\\",\\\"epru\\\":[],\\\"isDisabled\\\":false,\\\"maxDeposit\\\":\\\"1000000024671486719480691603261\\\",\\\"maxRedeem\\\":\\\"115792089237316195423570985008687907853269984665640564039457584007913129639935\\\"}\"}"

func getPool() *PoolSimulator {
	var poolE entity.Pool
	_ = json.Unmarshal([]byte(getPoolJSON), &poolE)
	pool, _ := NewPoolSimulator(poolE)
	return pool
}
func TestPoolSimulator_issue(t *testing.T) {
	t.Parallel()
	// https://etherscan.io/address/0x04b59F9F09750C044D7CfbC177561E409085f0f3#readContract
	// revert of getRequiredComponentIssuanceUnits
	p := getPool()
	assert.Equal(t, uint256.NewInt(1e18), p.getRequiredAmountSetToken(uint256.NewInt(1255394901774434543)))
	amountOut, err := p.CalcAmountOut(
		pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Amount: big.NewInt(1e18),
			},
			TokenOut: "0xc4506022fb8090774e8a628d5084eed61d9b99ee",
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(787175295419172397), amountOut.TokenAmountOut.Amount)
}

func TestPoolSimulator_redeem(t *testing.T) {
	t.Parallel()
	// https://etherscan.io/address/0x04b59F9F09750C044D7CfbC177561E409085f0f3#readContract
	// getRequiredComponentRedemptionUnits
	p := getPool()
	assert.Equal(t, uint256.NewInt(1255394901774434537), p.getRequiredComponentRedemptionUnits(uint256.NewInt(1e18)))
	amountOut, err := p.CalcAmountOut(
		pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xc4506022fb8090774e8a628d5084eed61d9b99ee",
				Amount: big.NewInt(1e18),
			},
			TokenOut: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(1270365070930608945), amountOut.TokenAmountOut.Amount)
}

const (
	testHyeth = "0xc4506022fb8090774e8a628d5084eed61d9b99ee"
	testWeth  = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
)

// newPoolWithEpru builds getPool's fixture with the given external position
// real units (decimal strings).
func newPoolWithEpru(t *testing.T, epru ...string) *PoolSimulator {
	t.Helper()
	var poolE entity.Pool
	src := getPoolJSON
	if len(epru) > 0 {
		src = strings.Replace(src, `\"epru\":[]`, `\"epru\":[\"`+strings.Join(epru, `\",\"`)+`\"]`, 1)
	}
	require.NoError(t, json.Unmarshal([]byte(src), &poolE))
	p, err := NewPoolSimulator(poolE)
	require.NoError(t, err)
	return p
}

func quote(p *PoolSimulator, in, out string, amt *big.Int) (*pool.CalcAmountOutResult, error) {
	return p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: in, Amount: amt},
		TokenOut:      out,
	})
}

// TestCalcAmountOut_RedeemRepeatable: quoting must not change the simulator,
// so the same redeem quote gives the same amount every time.
func TestCalcAmountOut_RedeemRepeatable(t *testing.T) {
	p := newPoolWithEpru(t, "1000000000000000", "0", "2500000000000000")
	unitBefore := p.defaultPositionRealUnit.String()
	first, err := quote(p, testHyeth, testWeth, big.NewInt(1e18))
	require.NoError(t, err)
	for range 3 {
		res, err := quote(p, testHyeth, testWeth, big.NewInt(1e18))
		require.NoError(t, err)
		assert.Equal(t, first.TokenAmountOut.Amount.String(), res.TokenAmountOut.Amount.String())
	}
	assert.Equal(t, unitBefore, p.defaultPositionRealUnit.String(), "quote wrote defaultPositionRealUnit")
}

// TestCalcAmountOut_ConcurrentSharedSimulator quotes one simulator from many
// goroutines; run with -race.
func TestCalcAmountOut_ConcurrentSharedSimulator(t *testing.T) {
	type job struct {
		in, out   string
		amt, want *big.Int
	}
	var jobs []job
	for i := range 16 {
		amt := new(big.Int).Mul(big.NewInt(int64(1+i*37)), big.NewInt(1e16))
		for _, d := range [][2]string{{testHyeth, testWeth}, {testWeth, testHyeth}} {
			res, err := quote(newPoolWithEpru(t, "1000000000000000"), d[0], d[1], amt)
			require.NoError(t, err)
			jobs = append(jobs, job{d[0], d[1], amt, res.TokenAmountOut.Amount})
		}
	}
	shared := newPoolWithEpru(t, "1000000000000000")
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 20 {
				for _, j := range jobs {
					res, err := quote(shared, j.in, j.out, j.amt)
					if !assert.NoError(t, err) || !assert.Equal(t, j.want.String(), res.TokenAmountOut.Amount.String()) {
						return
					}
				}
			}
		})
	}
	wg.Wait()
}

// relOff is |got - want| / want.
func relOff(want, got *big.Int) float64 {
	d := new(big.Float).SetInt(new(big.Int).Sub(got, want))
	r, _ := d.Quo(d, new(big.Float).SetInt(want)).Float64()
	return math.Abs(r)
}

// The fixture is the chain's state at block 22317092 (timestamp 1745235076): the vault 0x7019…e4ef's totalSupply
// and totalAssets, hyETH's totalSupply and its default position unit all equal the extra. Against that block's
// eth_call (2026-11-25):
//   - DebtIssuanceModule.getRequiredComponentRedemptionUnits(hyETH, 1e18) = 1255394901774434537, exactly;
//   - vault.previewDeposit(1e18) = 988216126278323508: the simulator's shares are 2.8e-7 under, so issuing never
//     over-quotes;
//   - vault.previewRedeem(1255394901774434537) = 1270364719206031425: the simulator's redeem is 2.8e-7 over
//     (0.003 bps). The cause of both gaps is not traced.
func TestRedeemAndDepositMatchChainAt22317092(t *testing.T) {
	t.Parallel()
	p := getPool()
	assert.Equal(t, uint256.NewInt(1255394901774434537), p.getRequiredComponentRedemptionUnits(uint256.NewInt(1e18)))

	shares, err := p.previewDeposit(uint256.NewInt(1e18))
	require.NoError(t, err)
	chainShares := big.NewInt(988216126278323508)
	assert.LessOrEqual(t, shares.ToBig().Cmp(chainShares), 0, "deposit shares above the chain's")
	assert.Less(t, relOff(chainShares, shares.ToBig()), 1e-6)

	assets, err := p.previewRedeem(uint256.NewInt(1255394901774434537))
	require.NoError(t, err)
	assert.Less(t, relOff(big.NewInt(1270364719206031425), assets.ToBig()), 1e-6)
}

// TestCalcAmountOut_RedeemAboveSupplyErrors: the fixture's hyETH supply is about 809e18 and hyETH holds about
// 1016e18 vault shares. Redeeming 1000e18 hyETH would burn more than the supply and redeem more shares than hyETH
// holds, which reverts on chain; the quote wrapped the post-swap supply and balance below zero instead.
func TestCalcAmountOut_RedeemAboveSupplyErrors(t *testing.T) {
	t.Parallel()
	p := getPool()
	_, err := quote(p, testHyeth, testWeth, new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18)))
	assert.ErrorIs(t, err, ErrRedeemMoreThanSupply)
	_, err = quote(p, testHyeth, testWeth, new(big.Int).Mul(big.NewInt(800), big.NewInt(1e18)))
	assert.NoError(t, err, "800e18 is within the supply and hyETH's shares")
}
