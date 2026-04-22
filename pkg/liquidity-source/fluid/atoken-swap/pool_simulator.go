package atokenswap

import (
	"math/big"
	"slices"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	Extra
	shortSymbols []string
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	prefixLen := len(entityPool.Tokens[0].Symbol) - 4
	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		Extra: extra,
		shortSymbols: lo.Map(entityPool.Tokens[1:], func(token *entity.PoolToken, _ int) string {
			return token.Symbol[prefixLen:]
		}),
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	// Check if contract is paused
	if s.Paused {
		return nil, ErrContractPaused
	}

	amountIn, overflow := uint256.FromBig(param.TokenAmountIn.Amount)
	if overflow || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	idxIn, idxOut := s.GetTokenIndex(param.TokenAmountIn.Token), s.GetTokenIndex(param.TokenOut)
	if idxIn == -1 || idxOut == -1 {
		return nil, ErrInvalidToken
	}

	// idxOut must be >= 1 (output tokens are at index 1, 2, ...)
	if idxOut < 1 || idxOut-1 >= len(s.OutputStates) {
		return nil, ErrInvalidToken
	}

	state := s.OutputStates[idxOut-1]

	// Check if amountIn exceeds maxSwap
	if amountIn.Cmp(state.MaxSwap) > 0 {
		return nil, ErrExcessiveSwapAmount
	}

	// Calculate amountOut using rate with premium
	amountOut, err := s.calculateAmountOut(amountIn, state.RateWithPremium, state.AvailableLiquidity)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: param.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            defaultGas,
		SwapInfo:       SwapInfo{ShortSymbol: s.shortSymbols[idxOut-1]},
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	// Check if contract is paused
	if s.Paused {
		return nil, ErrContractPaused
	}

	if param.TokenAmountOut.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmountOut
	}
	amountOut, overflow := uint256.FromBig(param.TokenAmountOut.Amount)
	if overflow {
		return nil, ErrInvalidAmountOut
	}

	idxIn, idxOut := s.GetTokenIndex(param.TokenIn), s.GetTokenIndex(param.TokenAmountOut.Token)
	if idxIn == -1 || idxOut == -1 {
		return nil, ErrInvalidToken
	}

	// idxOut must be >= 1 (output tokens are at index 1, 2, ...)
	if idxOut < 1 || idxOut-1 >= len(s.OutputStates) {
		return nil, ErrInvalidToken
	}

	state := s.OutputStates[idxOut-1]

	// Calculate amountIn using rate with premium (reverse calculation)
	amountIn, err := s.calculateAmountIn(amountOut, state.RateWithPremium, state.AvailableLiquidity)
	if err != nil {
		return nil, err
	}

	// Check if amountIn exceeds maxSwap
	if amountIn.Cmp(state.MaxSwap) > 0 {
		return nil, ErrExcessiveSwapAmount
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: param.TokenIn, Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: param.TokenAmountOut.Token, Amount: bignumber.ZeroBI},
		Gas:           defaultGas,
		SwapInfo:      SwapInfo{ShortSymbol: s.shortSymbols[idxOut-1]},
	}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	inputAmount, outputAmount := input.Amount, output.Amount

	// Find which output token was swapped
	idxOut := s.GetTokenIndex(output.Token)
	if idxOut < 1 || idxOut-1 >= len(s.OutputStates) {
		return // Invalid output token index
	}

	// Convert amounts to uint256
	inputUint256, overflow := uint256.FromBig(inputAmount)
	if overflow {
		return
	}
	outputUint256, overflow := uint256.FromBig(outputAmount)
	if overflow {
		return
	}

	s.OutputStates = slices.Clone(s.OutputStates)
	state := &s.OutputStates[idxOut-1]

	// Update AvailableLiquidity (subtract output amount)
	state.AvailableLiquidity = outputUint256.Sub(state.AvailableLiquidity, outputUint256)

	// Update MaxSwap (subtract input amount)
	state.MaxSwap = inputUint256.Sub(state.MaxSwap, inputUint256)
}

func (s *PoolSimulator) CanSwapTo(address string) []string {
	// Can swap TO any output token from the input token
	idx := s.GetTokenIndex(address)
	if idx < 1 {
		return nil
	}
	// Output tokens are at indices 1, 2, ...
	return []string{s.Info.Tokens[0]} // Can swap from input to this output
}

func (s *PoolSimulator) CanSwapFrom(address string) []string {
	// Can swap FROM input token to any output token
	idx := s.GetTokenIndex(address)
	// Input token is at index 0
	if idx == 0 {
		return s.Info.Tokens[1:] // Can swap from input to all outputs
	}
	// Can't swap FROM output tokens
	return nil
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{
		BlockNumber: s.Info.BlockNumber,
	}
}

// calculateAmountOut calculates output amount for a given input amount
// Formula: amountOut = (amountIn * 1e18) / rateWithPremium
func (s *PoolSimulator) calculateAmountOut(amountIn, rateWithPremium, availableLiquidity *uint256.Int) (*uint256.Int,
	error) {
	// Calculate amountOut using rate with premium
	// amountOut = (amountIn * 1e18) / rateWithPremium
	amountOut := big256.MulDiv(amountIn, big256.BONE, rateWithPremium)

	// Check if amountOut exceeds available liquidity
	if amountOut.Cmp(availableLiquidity) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	return amountOut, nil
}

// calculateAmountIn calculates input amount for a given output amount
// Formula: amountIn = (amountOut * rateWithPremium) / 1e18
func (s *PoolSimulator) calculateAmountIn(amountOut, rateWithPremium, availableLiquidity *uint256.Int) (*uint256.Int,
	error) {
	// Check if amountOut exceeds available liquidity
	if amountOut.Cmp(availableLiquidity) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	// Calculate amountIn using rate with premium
	// amountIn = (amountOut * rateWithPremium) / 1e18
	amountIn := big256.MulDiv(amountOut, rateWithPremium, big256.BONE)

	return amountIn, nil
}
