package stabull

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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

	tokens := make([]string, 2)
	reserves := make([]*big.Int, 2)

	if len(entityPool.Reserves) == 2 && len(entityPool.Tokens) == 2 {
		tokens[0] = entityPool.Tokens[0].Address
		reserves[0] = bignumber.NewBig10(entityPool.Reserves[0])

		tokens[1] = entityPool.Tokens[1].Address
		reserves[1] = bignumber.NewBig10(entityPool.Reserves[1])
	}

	info := pool.PoolInfo{
		Address:  strings.ToLower(entityPool.Address),
		SwapFee:  big.NewInt(0), // Stabull doesn't have explicit swap fee (built into curve)
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
// 4. 0.15% swap fee (70% to LPs, 30% to protocol) built into calculation
//
// Since we're in a simulator context (no RPC calls), we implement a simplified version
// For production routing, the pool_tracker ensures we have up-to-date reserves and parameters
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

	// Simplified Stabull curve calculation
	// Real implementation in Curve.sol is much more sophisticated
	// It uses viewOriginSwap which:
	// - Calculates new reserve ratio after swap
	// - Applies curve formula with alpha, beta, delta, epsilon, lambda
	// - Adjusts pricing based on oracle rate deviation
	// - Applies 0.15% fee
	//
	// For accurate routing, we use a constant product approximation with fee
	// This gives reasonable estimates; actual execution uses on-chain viewOriginSwap

	// Apply 0.15% swap fee (15 basis points)
	feeAmount := new(big.Int).Mul(amountIn, big.NewInt(swapFeeBps))
	feeAmount = new(big.Int).Div(feeAmount, big.NewInt(10000))

	amountInAfterFee := new(big.Int).Sub(amountIn, feeAmount)

	// Constant product approximation: amountOut = (reserveOut * amountInAfterFee) / (reserveIn + amountInAfterFee)
	numerator := new(big.Int).Mul(reserveOut, amountInAfterFee)
	denominator := new(big.Int).Add(reserveIn, amountInAfterFee)

	if denominator.Cmp(bignumber.ZeroBI) == 0 {
		return nil, fmt.Errorf("zero denominator in swap calculation")
	}

	amountOut := new(big.Int).Div(numerator, denominator)

	// TODO: For more accurate simulation, implement full Stabull curve formula
	// This would require:
	// 1. Parse viewOriginSwap logic from Curve.sol
	// 2. Apply curve parameters from p.extra.CurveParams
	// 3. Integrate oracle rate adjustment if available
	//
	// For now, constant product with 0.15% fee provides reasonable routing estimates

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
