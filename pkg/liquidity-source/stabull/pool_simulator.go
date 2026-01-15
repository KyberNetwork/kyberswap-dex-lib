package stabull

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/logger"
)

type PoolSimulator struct {
	pool.Pool
	gas   Gas
	extra Extra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	// Debug: Log what Extra we received
	logger.WithFields(logger.Fields{
		"dex":             DexType,
		"pool":            entityPool.Address,
		"baseOracle":      extra.BaseOracleAddress,
		"quoteOracle":     extra.QuoteOracleAddress,
		"baseOracleRate":  extra.BaseOracleRate,
		"quoteOracleRate": extra.QuoteOracleRate,
	}).Debug("NewPoolSimulator created")

	tokens := make([]string, 2)
	reserves := make([]*big.Int, 2)

	if len(entityPool.Reserves) == 2 && len(entityPool.Tokens) == 2 {
		tokens[0] = entityPool.Tokens[0].Address
		reserves[0] = bignumber.NewBig10(entityPool.Reserves[0])

		tokens[1] = entityPool.Tokens[1].Address
		reserves[1] = bignumber.NewBig10(entityPool.Reserves[1])
	}

	// Derive swap fee from epsilon parameter
	var swapFee *big.Int
	if epsilon, ok := new(big.Int).SetString(extra.CurveParams.Epsilon, 10); ok && epsilon != nil {
		// Epsilon is the fee parameter (e.g., 1.5e15 for 0.15%)
		// We store it as the swap fee in the pool info
		swapFee = epsilon
	} else {
		// Default to 0.15% = 1.5e15 in 1e18 precision
		swapFee = new(big.Int).Mul(big.NewInt(15), big.NewInt(1e14))
	}

	info := pool.PoolInfo{
		Address:  strings.ToLower(entityPool.Address),
		SwapFee:  swapFee, // Fee derived from epsilon parameter
		Exchange: entityPool.Exchange,
		Type:     entityPool.Type,
		Tokens:   tokens,
		Reserves: reserves,
	}

	return &PoolSimulator{
		Pool:  pool.Pool{Info: info},
		gas:   defaultGas,
		extra: extra,
	}, nil
}

// CalcAmountOut calculates the expected output amount for a given input
// Uses cached reserve and curve parameter state from pool tracker
func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut

	tokenInIndex := p.GetTokenIndex(tokenAmountIn.Token)
	tokenOutIndex := p.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, fmt.Errorf("tokenInIndex: %v or tokenOutIndex: %v is not correct", tokenInIndex, tokenOutIndex)
	}

	// Calculate swap using Stabull curve formula
	// Note: The actual contract has viewOriginSwap(origin, target, originAmount) that returns targetAmount
	// In the simulator, we need to replicate this logic locally using cached curve parameters
	amountOut, err := p.calculateSwap(
		tokenAmountIn.Amount,
		tokenInIndex,
		tokenOutIndex,
	)
	if err != nil {
		return nil, err
	}

	// Check if we have enough reserves
	if amountOut.Cmp(p.Info.Reserves[tokenOutIndex]) >= 0 {
		return nil, fmt.Errorf("insufficient reserves: need %s, have %s",
			amountOut.String(), p.Info.Reserves[tokenOutIndex].String())
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: bignumber.ZeroBI, // Fee is built into amountOut calculation
		},
		Gas: p.gas.Swap,
	}, nil
}

// calculateSwap implements the Stabull curve swap calculation logic
// Stabull uses a sophisticated invariant-based curve with oracle integration
// The actual contract uses viewOriginSwap(origin, target, amount) which implements:
// 1. Hybrid constant product and constant sum invariant
// 2. Dynamic pricing based on pool balance vs oracle rate
// 3. Curve parameters (alpha, beta, delta, epsilon, lambda) define the shape
// 4. Dynamic fee based on epsilon and pool imbalance
//
// We implement the curve math using the greek parameters from pool state
func (p *PoolSimulator) calculateSwap(
	amountIn *big.Int,
	tokenInIndex int,
	tokenOutIndex int,
) (*big.Int, error) {
	if amountIn == nil || amountIn.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, fmt.Errorf("invalid amount in")
	}

	reserveIn := p.Info.Reserves[tokenInIndex]
	reserveOut := p.Info.Reserves[tokenOutIndex]

	if reserveIn == nil || reserveIn.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, fmt.Errorf("insufficient reserve in")
	}

	if reserveOut == nil || reserveOut.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, fmt.Errorf("insufficient reserve out")
	}

	// Parse curve parameters from extra
	alpha, ok := new(big.Int).SetString(p.extra.CurveParams.Alpha, 10)
	if !ok || alpha == nil {
		alpha = new(big.Int).Mul(big.NewInt(5), big.NewInt(1e17)) // Default: 0.5 * 1e18
	}

	beta, ok := new(big.Int).SetString(p.extra.CurveParams.Beta, 10)
	if !ok || beta == nil {
		beta = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) // Default: 1e18
	}

	delta, ok := new(big.Int).SetString(p.extra.CurveParams.Delta, 10)
	if !ok || delta == nil {
		delta = new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil) // Default: 0.1 * 1e18
	}

	epsilon, ok := new(big.Int).SetString(p.extra.CurveParams.Epsilon, 10)
	if !ok || epsilon == nil {
		epsilon = new(big.Int).Mul(big.NewInt(15), big.NewInt(1e14)) // Default: 0.15% = 1.5e15
	}

	lambda, ok := new(big.Int).SetString(p.extra.CurveParams.Lambda, 10)
	if !ok || lambda == nil {
		lambda = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) // Default: 1e18
	}

	// Convert amountIn to numeraire using the input token's oracle rate
	// The contract's Assimilators convert token amounts to a common numeraire (USD)
	// Reserves are stored in numeraire terms
	var inputOracleRate *big.Int
	if tokenInIndex == 0 {
		// Swapping base token (token0) → quote token (token1)
		if p.extra.BaseOracleRate != "" {
			inputOracleRate, _ = new(big.Int).SetString(p.extra.BaseOracleRate, 10)
		}
	} else {
		// Swapping quote token (token1) → base token (token0)
		if p.extra.QuoteOracleRate != "" {
			inputOracleRate, _ = new(big.Int).SetString(p.extra.QuoteOracleRate, 10)
		}
	}

	// Validate that we have the required oracle rate
	if inputOracleRate == nil || inputOracleRate.Cmp(bignumber.ZeroBI) <= 0 {
		if tokenInIndex == 0 {
			return nil, fmt.Errorf("missing or invalid BaseOracleRate for input token conversion (have: '%s')", p.extra.BaseOracleRate)
		}
		return nil, fmt.Errorf("missing or invalid QuoteOracleRate for input token conversion (have: '%s', addr: '%s')", p.extra.QuoteOracleRate, p.extra.QuoteOracleAddress)
	}

	// Convert input to numeraire: (amountIn * oracleRate) / 1e8
	// Chainlink oracle rates are 8 decimals
	amountInNumeraire := new(big.Int).Mul(amountIn, inputOracleRate)
	amountInNumeraire.Div(amountInNumeraire, big.NewInt(1e8))

	// Parse oracle rate from extra (for internal curve pricing)
	var oracleRate *big.Int
	if p.extra.OracleRate != "" {
		oracleRate, _ = new(big.Int).SetString(p.extra.OracleRate, 10)
	}

	// Use the Stabull curve formula with greek parameters
	amountOutNumeraire, err := calculateStabullSwap(
		amountInNumeraire, // Use numeraire-adjusted amount
		reserveIn,
		reserveOut,
		alpha,
		beta,
		delta,
		epsilon,
		lambda,
		oracleRate, // Pass oracle rate for pricing adjustment
	)
	if err != nil {
		return nil, err
	}

	// Convert output from numeraire to actual token amount
	// For base→quote (token0→token1): output is already in quote token units (numeraire)
	// For quote→base (token1→token0): output is in numeraire, need to convert to base token units
	amountOut := amountOutNumeraire

	if tokenOutIndex == 0 {
		// Swapping to base token (token0), need to convert from numeraire
		var outputOracleRate *big.Int
		if p.extra.BaseOracleRate != "" {
			outputOracleRate, _ = new(big.Int).SetString(p.extra.BaseOracleRate, 10)
		}

		// Validate that we have the required oracle rate for output conversion
		if outputOracleRate == nil || outputOracleRate.Cmp(bignumber.ZeroBI) <= 0 {
			return nil, fmt.Errorf("missing or invalid BaseOracleRate for output token conversion")
		}

		// Convert from numeraire to token: (amountOutNumeraire * 1e8) / oracleRate
		amountOut = new(big.Int).Mul(amountOutNumeraire, big.NewInt(1e8))
		amountOut.Div(amountOut, outputOracleRate)
	}
	// For tokenOutIndex == 1 (quote token), output is already in correct units

	return amountOut, nil
}

// UpdateBalance updates the pool state after a swap
func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut

	for i := range p.Info.Tokens {
		if p.Info.Tokens[i] == input.Token {
			// Add input amount to reserves
			p.Info.Reserves[i] = new(big.Int).Add(p.Info.Reserves[i], input.Amount)
		}
		if p.Info.Tokens[i] == output.Token {
			// Subtract output amount from reserves
			p.Info.Reserves[i] = new(big.Int).Sub(p.Info.Reserves[i], output.Amount)
		}
	}
}

// GetMetaInfo returns metadata about the pool
func (p *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	meta := Meta{
		Alpha:   p.extra.CurveParams.Alpha,
		Beta:    p.extra.CurveParams.Beta,
		Delta:   p.extra.CurveParams.Delta,
		Epsilon: p.extra.CurveParams.Epsilon,
		Lambda:  p.extra.CurveParams.Lambda,
	}

	if p.extra.OracleRate != "" {
		meta.OracleRate = p.extra.OracleRate
	}

	return meta
}

// CanSwapTo checks if a swap to the given address is possible
func (p *PoolSimulator) CanSwapTo(address string) []string {
	// Return list of tokens that can be swapped to the given token
	for i, token := range p.Info.Tokens {
		if strings.EqualFold(token, address) {
			// Can swap to the other token in the pair
			otherIndex := 1 - i
			return []string{p.Info.Tokens[otherIndex]}
		}
	}
	return nil
}

// CanSwapFrom checks if a swap from the given address is possible
func (p *PoolSimulator) CanSwapFrom(address string) []string {
	return p.CanSwapTo(address)
}

// GetLpToken returns the LP token address
// Stabull pools are ERC20 tokens themselves (LP tokens)
func (p *PoolSimulator) GetLpToken() string {
	// The pool address itself is the LP token
	return p.Info.Address
}
