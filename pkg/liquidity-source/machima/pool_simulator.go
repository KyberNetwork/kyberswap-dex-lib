package machima

import (
	"math/big"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// PoolSimulator wraps the UniV3 pool simulator and applies Machima's tax layer.
type PoolSimulator struct {
	*uniswapv3.PoolSimulator

	BuyTaxBps          uint16
	SellTaxBps         uint16
	HasTax             bool
	CounterAsset       string
	Token              string
	PoolDeploymentTime uint64
	RouterAddress      string
	// Global counter-asset set (lowercased)
	WETH string
	USDC string
	XMA  string
	// XMA sell price floor
	XmaSellSqrtPriceLimit *uint256.Int
}

// Factory registration — Kyber's pool infrastructure picks this up automatically
var _ = pool.RegisterFactory1(DexTypeMachima, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	// Build a UniV3 ExtraTickU256 from the Machima extra to delegate tick math
	v3Extra := uniswapv3.ExtraTickU256{
		SqrtPriceX96: extra.SqrtPriceX96,
		Tick:         extra.Tick,
		Liquidity:    extra.Liquidity,
		TickSpacing:  uint64(extra.TickSpacing),
	}

	// Convert our ticks to UniV3 format
	v3Ticks := make([]uniswapv3.TickU256, 0, len(extra.Ticks))
	for _, t := range extra.Ticks {
		v3Ticks = append(v3Ticks, uniswapv3.TickU256{
			Index:          t.Index,
			LiquidityGross: t.LiquidityGross,
			LiquidityNet:   t.LiquidityNet,
		})
	}
	v3Extra.Ticks = v3Ticks

	v3Sim, err := uniswapv3.NewPoolSimulatorWithExtra(entityPool, &v3Extra, uniswapv3.SimulatorConfig{
		AllowEmptyTicks: true,
	})
	if err != nil {
		return nil, err
	}

	return &PoolSimulator{
		PoolSimulator:         v3Sim,
		BuyTaxBps:             extra.BuyTaxBps,
		SellTaxBps:            extra.SellTaxBps,
		HasTax:                extra.HasTax,
		CounterAsset:          extra.CounterAsset,
		Token:                 extra.Token,
		PoolDeploymentTime:    extra.PoolDeploymentTime,
		RouterAddress:         staticExtra.RouterAddress,
		WETH:                  strings.ToLower(staticExtra.WETH),
		USDC:                  strings.ToLower(staticExtra.USDC),
		XMA:                   strings.ToLower(staticExtra.XMA),
		XmaSellSqrtPriceLimit: extra.XmaSellSqrtPriceLimit,
	}, nil
}

// CalcAmountOut computes expected output with Machima's tax applied.
// Buy (counter→token): tax on input, then V3 swap.
// Sell (token→counter): V3 swap, then tax on output.
func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if p.isInAntiSniperWindow() {
		return nil, ErrAntiSniperActive
	}

	tokenIn := param.TokenAmountIn.Token
	tokenOut := param.TokenOut
	amountIn := param.TokenAmountIn.Amount

	_, _, isBuy, err := p.classifyPair(tokenIn, tokenOut)
	if err != nil {
		return nil, err
	}

	if isBuy {
		return p.calcBuyAmountOut(tokenIn, tokenOut, amountIn)
	}
	return p.calcSellAmountOut(tokenIn, tokenOut, amountIn)
}

func (p *PoolSimulator) calcBuyAmountOut(tokenIn, tokenOut string, amountIn *big.Int) (*pool.CalcAmountOutResult, error) {
	// Buy: tax deducted from input first, remainder goes to pool
	swapAmount := new(big.Int).Set(amountIn)

	if p.HasTax && p.BuyTaxBps > 0 {
		taxAmount := new(big.Int).Mul(amountIn, big.NewInt(int64(p.BuyTaxBps)))
		taxAmount.Div(taxAmount, big.NewInt(10000))
		swapAmount.Sub(amountIn, taxAmount)
	}

	result, err := p.PoolSimulator.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: swapAmount},
		TokenOut:      tokenOut,
	})
	if err != nil {
		return nil, err
	}

	result.Gas = BaseGas + CrossTickGas
	return result, nil
}

func (p *PoolSimulator) calcSellAmountOut(tokenIn, tokenOut string, amountIn *big.Int) (*pool.CalcAmountOutResult, error) {
	// When selling XMA, apply xmaSellSqrtPriceLimit as the price floor.
	// This mirrors the on-chain quoter which passes it to QuoterV2 as sqrtPriceLimitX96.
	useFloor := strings.EqualFold(tokenIn, p.XMA) && p.XmaSellSqrtPriceLimit != nil && !p.XmaSellSqrtPriceLimit.IsZero()

	var result *pool.CalcAmountOutResult
	var err error

	if useFloor {
		// Call V3Pool directly with the price limit
		tokenInIndex := p.GetTokenIndex(tokenIn)
		tokenOutIndex := p.GetTokenIndex(tokenOut)
		if tokenInIndex < 0 || tokenOutIndex < 0 {
			return nil, ErrInvalidPair
		}

		var amountInU uint256.Int
		if overflow := amountInU.SetFromBig(amountIn); overflow {
			return nil, ErrInvalidPair
		}

		zeroForOne := tokenInIndex == 0
		v3Result, v3Err := p.V3Pool.GetOutputAmountV2(zeroForOne, amountInU, *p.XmaSellSqrtPriceLimit)
		if v3Err != nil {
			return nil, v3Err
		}

		amountOutBI := v3Result.AmountCalculated.ToBig()
		if amountOutBI.Cmp(p.GetReserves()[tokenOutIndex]) > 0 {
			return nil, uniswapv3.ErrInsufficientBalance
		}

		remainingAmount := big.NewInt(0)
		if !v3Result.RemainingAmountIn.IsZero() {
			remainingAmount = v3Result.RemainingAmountIn.ToBig()
		}

		result = &pool.CalcAmountOutResult{
			TokenAmountOut:         &pool.TokenAmount{Token: tokenOut, Amount: amountOutBI},
			RemainingTokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: remainingAmount},
			Fee:                    &pool.TokenAmount{Token: tokenIn, Amount: big.NewInt(0)},
			Gas:                    BaseGas + CrossTickGas,
		}
	} else {
		// Standard path: no price floor
		result, err = p.PoolSimulator.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
			TokenOut:      tokenOut,
		})
		if err != nil {
			return nil, err
		}
	}

	// Apply sell tax on output
	if p.HasTax && p.SellTaxBps > 0 {
		rawOutput := result.TokenAmountOut.Amount
		taxAmount := new(big.Int).Mul(rawOutput, big.NewInt(int64(p.SellTaxBps)))
		taxAmount.Div(taxAmount, big.NewInt(10000))
		result.TokenAmountOut.Amount = new(big.Int).Sub(rawOutput, taxAmount)
	}

	result.Gas = BaseGas + CrossTickGas
	return result, nil
}

func (p *PoolSimulator) isInAntiSniperWindow() bool {
	if p.PoolDeploymentTime == 0 {
		return false
	}
	return uint64(time.Now().Unix()) < p.PoolDeploymentTime+AntiSniperWindowSeconds
}

// classifyPair mirrors on-chain _classifyPair logic using the global counter-asset set.
// Returns (token, counterAsset, isBuy, error).
func (p *PoolSimulator) classifyPair(tokenIn, tokenOut string) (string, string, bool, error) {
	inIsCounter := p.isGlobalCounterAsset(tokenIn)
	outIsCounter := p.isGlobalCounterAsset(tokenOut)

	if inIsCounter && !outIsCounter {
		return tokenOut, tokenIn, true, nil
	} else if !inIsCounter && outIsCounter {
		return tokenIn, tokenOut, false, nil
	} else if inIsCounter && outIsCounter {
		// Both are counter-assets — only valid when exactly one side is XMA
		if strings.EqualFold(tokenIn, p.XMA) && !strings.EqualFold(tokenOut, p.XMA) {
			return p.XMA, tokenOut, false, nil // selling XMA
		} else if strings.EqualFold(tokenOut, p.XMA) && !strings.EqualFold(tokenIn, p.XMA) {
			return p.XMA, tokenIn, true, nil // buying XMA
		}
		return "", "", false, ErrInvalidPair
	}
	return "", "", false, ErrInvalidPair
}

func (p *PoolSimulator) isGlobalCounterAsset(token string) bool {
	return strings.EqualFold(token, p.WETH) ||
		strings.EqualFold(token, p.USDC) ||
		strings.EqualFold(token, p.XMA)
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return PoolMeta{
		Router:   p.RouterAddress,
		Deadline: uint64(time.Now().Unix()) + 300,
	}
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	v3Clone := p.PoolSimulator.CloneState().(*uniswapv3.PoolSimulator)
	cloned.PoolSimulator = v3Clone
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	p.PoolSimulator.UpdateBalance(params)
}

// Ensure interface compliance
var _ pool.IPoolSimulator = (*PoolSimulator)(nil)
