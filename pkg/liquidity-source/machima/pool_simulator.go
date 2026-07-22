package machima

import (
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// PoolSimulator wraps the UniV3 simulator and layers Machima's tax, pair-classification and
// anti-sniper rules on top.
//
// The embedded simulator is what makes the tick math available, but every method that the tax
// layer changes the result of MUST be overridden here — a promoted method silently bypasses tax.
// That currently means CalcAmountOut, CalcAmountIn, UpdateBalance, CloneState and GetMetaInfo.
type PoolSimulator struct {
	*uniswapv3.PoolSimulator

	buyTaxBps          uint16
	sellTaxBps         uint16
	hasTax             bool
	poolDeploymentTime uint64
	routerAddress      string

	// Global counter-asset set, lowercased once at construction.
	weth string
	usdc string
	xma  string

	// xmaSellSqrtPriceLimit is the launch-tick floor applied when XMA is sold. Zero means no floor.
	xmaSellSqrtPriceLimit uint256.Int
}

var (
	_ pool.IPoolSimulator         = (*PoolSimulator)(nil)
	_ pool.IPoolExactOutSimulator = (*PoolSimulator)(nil)

	_ = pool.RegisterFactory1(DexType, NewPoolSimulator)
)

func NewPoolSimulator(entityPool entity.Pool, _ valueobject.ChainID) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	if extra.BuyTaxBps >= bpsDenominator || extra.SellTaxBps >= bpsDenominator {
		return nil, ErrTaxTooHigh
	}

	// Decode the same Extra a second time into the UniV3 tick-math shape. Both views share the
	// JSON field names; uint256/int256 accept the plain numbers big.Int writes.
	var v3Extra uniswapv3.ExtraTickU256
	if err := json.Unmarshal([]byte(entityPool.Extra), &v3Extra); err != nil {
		return nil, err
	}

	v3Sim, err := uniswapv3.NewPoolSimulatorWithExtra(entityPool, &v3Extra, uniswapv3.SimulatorConfig{})
	if err != nil {
		return nil, err
	}

	// A Machima swap goes through the aggregator router rather than straight to the pool, so it
	// costs more than the bare UniV3 swap the embedded simulator would otherwise assume.
	v3Sim.Gas = defaultGas

	sim := &PoolSimulator{
		PoolSimulator:      v3Sim,
		buyTaxBps:          extra.BuyTaxBps,
		sellTaxBps:         extra.SellTaxBps,
		hasTax:             extra.HasTax,
		poolDeploymentTime: extra.PoolDeploymentTime,
		routerAddress:      staticExtra.RouterAddress,
		weth:               strings.ToLower(staticExtra.WETH),
		usdc:               strings.ToLower(staticExtra.USDC),
		xma:                strings.ToLower(staticExtra.XMA),
	}
	if extra.XmaSellSqrtPriceLimit != nil {
		if overflow := sim.xmaSellSqrtPriceLimit.SetFromBig(extra.XmaSellSqrtPriceLimit); overflow {
			return nil, ErrOverflow
		}
	}
	return sim, nil
}

// CalcAmountOut applies Machima's tax layer around the V3 swap:
//   - buy (counter asset -> token): tax is taken off the input, the remainder is swapped
//   - sell (token -> counter asset): the V3 output is swapped first, then tax is taken off it
func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if p.isInAntiSniperWindow() {
		return nil, ErrAntiSniperActive
	}

	tokenIn, tokenOut := param.TokenAmountIn.Token, param.TokenOut
	isBuy, err := p.classifyPair(tokenIn, tokenOut)
	if err != nil {
		return nil, err
	}

	amountIn := param.TokenAmountIn.Amount
	poolAmountIn := amountIn
	if isBuy {
		poolAmountIn = deductBps(amountIn, p.taxBps(true))
	}
	if poolAmountIn.Sign() <= 0 {
		return nil, ErrZeroAmount
	}

	res, err := p.PoolSimulator.CalcAmountOutWithPriceLimit(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: poolAmountIn},
		TokenOut:      tokenOut,
	}, p.sqrtPriceLimit(tokenIn, isBuy))
	if err != nil {
		return nil, err
	}

	v3SwapInfo, ok := res.SwapInfo.(uniswapv3.SwapInfo)
	if !ok {
		return nil, ErrUnexpectedSwapInfo
	}

	poolAmountOut := res.TokenAmountOut.Amount
	amountOut := poolAmountOut
	if !isBuy {
		amountOut = deductBps(poolAmountOut, p.taxBps(false))
	}
	if amountOut.Sign() <= 0 {
		return nil, ErrZeroAmount
	}

	// Buy tax is charged on the full amountIn, not just the portion the pool consumes, so a
	// partial fill refunds the leftover post-tax input as-is with no tax rebate. Verified against
	// MachimaAggregatorQuoter.quote(): taxAmount stays exactly buyTaxBps of amountIn even at sizes
	// far past the pool's liquidity, where the swap only partially fills.
	remaining := res.RemainingTokenAmountIn.Amount

	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: tokenOut, Amount: amountOut},
		RemainingTokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: remaining},
		Fee:                    &pool.TokenAmount{Token: tokenIn, Amount: new(big.Int).Sub(amountIn, poolAmountIn)},
		Gas:                    res.Gas,
		SwapInfo: SwapInfo{
			V3: v3SwapInfo,
			// Fresh values: with a zero tax the amounts above alias the caller's and the V3 leg's
			// big.Ints, and SwapInfo outlives the result that UpdateBalance is called with.
			PoolAmountIn:  new(big.Int).Sub(poolAmountIn, remaining),
			PoolAmountOut: new(big.Int).Set(poolAmountOut),
		},
	}, nil
}

// CalcAmountIn inverts CalcAmountOut. It matters even though the pathfinder does not use exact-out:
// onchain-price-service's ApproxAmountIn prefers a simulator's own CalcAmountIn over its
// regula-falsi fallback, then re-checks the answer with CalcAmountOut and drops the pool when the
// two disagree. An untaxed CalcAmountIn would therefore delete Machima pools from the price graph.
func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if p.isInAntiSniperWindow() {
		return nil, ErrAntiSniperActive
	}

	tokenIn, tokenOut := param.TokenIn, param.TokenAmountOut.Token
	isBuy, err := p.classifyPair(tokenIn, tokenOut)
	if err != nil {
		return nil, err
	}

	// Gross the requested output back up to what the pool has to produce before sell tax.
	poolAmountOut := param.TokenAmountOut.Amount
	if !isBuy {
		poolAmountOut = grossUpBps(poolAmountOut, p.taxBps(false))
	}
	if poolAmountOut.Sign() <= 0 {
		return nil, ErrZeroAmount
	}

	res, err := p.PoolSimulator.CalcAmountInWithPriceLimit(pool.CalcAmountInParams{
		TokenIn:        tokenIn,
		TokenAmountOut: pool.TokenAmount{Token: tokenOut, Amount: poolAmountOut},
	}, p.sqrtPriceLimit(tokenIn, isBuy))
	if err != nil {
		return nil, err
	}

	v3SwapInfo, ok := res.SwapInfo.(uniswapv3.SwapInfo)
	if !ok {
		return nil, ErrUnexpectedSwapInfo
	}

	// Gross the pool-side input up to what the user has to send before buy tax.
	poolAmountIn := res.TokenAmountIn.Amount
	amountIn := poolAmountIn
	if isBuy {
		amountIn = grossUpBps(poolAmountIn, p.taxBps(true))
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: amountIn},
		Fee:           &pool.TokenAmount{Token: tokenIn, Amount: new(big.Int).Sub(amountIn, poolAmountIn)},
		Gas:           res.Gas,
		SwapInfo: SwapInfo{
			V3:            v3SwapInfo,
			PoolAmountIn:  new(big.Int).Set(poolAmountIn),
			PoolAmountOut: new(big.Int).Set(poolAmountOut),
		},
	}, nil
}

// UpdateBalance forwards the pool-side amounts, not the user-facing ones: tax never reaches the
// pool, so applying the taxed amounts to reserves would drift them on every update.
func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warnf("failed to UpdateBalance for machima pool %s, wrong swapInfo type", p.GetAddress())
		return
	}

	p.PoolSimulator.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: si.PoolAmountIn},
		TokenAmountOut: pool.TokenAmount{Token: params.TokenAmountOut.Token, Amount: si.PoolAmountOut},
		SwapInfo:       si.V3,
	})
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.PoolSimulator = p.PoolSimulator.CloneState().(*uniswapv3.PoolSimulator)
	return &cloned
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return PoolMeta{Router: p.routerAddress, ApprovalAddress: p.routerAddress}
}

// GetApprovalAddress returns the aggregator router, which is what the executor approves before
// calling IMachima(router).swap.
func (p *PoolSimulator) GetApprovalAddress(_ string, _ string) string {
	return p.routerAddress
}

// sqrtPriceLimit returns the launch-tick floor the swap adapter pins for XMA sells. Every other
// direction swaps without a limit.
//
// The adapter exposes one global floor while sqrtPriceX96 is pool-specific, so this is only sound
// because the floor can only ever reach one pool: it applies to sells of XMA, and a sell means XMA
// is the traded token, which it only is in the XMA/WETH pool. Everywhere else XMA is a counter
// asset, so sending XMA in classifies as a buy and takes the unlimited branch.
func (p *PoolSimulator) sqrtPriceLimit(tokenIn string, isBuy bool) uint256.Int {
	if isBuy || tokenIn != p.xma {
		return uint256.Int{}
	}
	return p.xmaSellSqrtPriceLimit
}

func (p *PoolSimulator) taxBps(isBuy bool) uint16 {
	if !p.hasTax {
		return 0
	}
	if isBuy {
		return p.buyTaxBps
	}
	return p.sellTaxBps
}

func (p *PoolSimulator) isInAntiSniperWindow() bool {
	if p.poolDeploymentTime == 0 {
		return false
	}
	return uint64(time.Now().Unix()) < p.poolDeploymentTime+AntiSniperWindowSeconds
}

// classifyPair mirrors the router's _classifyPair. Exactly one side must be the traded token; XMA
// is special because it is both a counter asset for other tokens and the traded token of its own
// XMA/WETH pool.
func (p *PoolSimulator) classifyPair(tokenIn, tokenOut string) (isBuy bool, err error) {
	inIsCounter, outIsCounter := p.isCounterAsset(tokenIn), p.isCounterAsset(tokenOut)

	switch {
	case inIsCounter && !outIsCounter:
		return true, nil
	case !inIsCounter && outIsCounter:
		return false, nil
	case inIsCounter && outIsCounter:
		// Both sides are counter assets: only valid when exactly one of them is XMA, which is
		// then the traded token.
		if tokenIn == p.xma && tokenOut != p.xma {
			return false, nil // selling XMA
		}
		if tokenOut == p.xma && tokenIn != p.xma {
			return true, nil // buying XMA
		}
	}
	return false, ErrInvalidPair
}

func (p *PoolSimulator) isCounterAsset(token string) bool {
	return token == p.weth || token == p.usdc || token == p.xma
}

// deductBps returns amount - floor(amount*bps/10000), matching the contract's rounding.
func deductBps(amount *big.Int, bps uint16) *big.Int {
	if bps == 0 {
		return amount
	}
	tax := new(big.Int).Mul(amount, big.NewInt(int64(bps)))
	tax.Div(tax, bigBpsDenominator)
	return tax.Sub(amount, tax)
}

// grossUpBps is the inverse of deductBps, rounded up so the grossed-up amount always clears the
// target after the contract's flooring tax.
func grossUpBps(amount *big.Int, bps uint16) *big.Int {
	if bps == 0 {
		return amount
	}
	gross := new(big.Int).Mul(amount, bigBpsDenominator)
	den := big.NewInt(int64(bpsDenominator - bps))
	gross.Add(gross, new(big.Int).Sub(den, bignumber.One))
	return gross.Div(gross, den)
}
