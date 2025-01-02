package math

import (
	"errors"
	"sync"

	"github.com/holiman/uint256"
)

var (
	ErrStableInvariantDidNotConverge      = errors.New("stable invariant didn't converge")
	ErrStableComputeBalanceDidNotConverge = errors.New("stable computeBalance didn't converge")

	_AMP_PRECISION = uint256.NewInt(1e3)
)

var StableMath *stableMath

type stableMath struct {
	mu sync.RWMutex
}

func init() {
	StableMath = &stableMath{}
}

func (s *stableMath) ComputeOutGivenExactIn(
	amplificationParameter *uint256.Int,
	balances []*uint256.Int,
	tokenIndexIn, tokenIndexOut int,
	tokenAmountIn, invariant *uint256.Int,
) (amountOut *uint256.Int, err error) {
	/**************************************************************************************************************
	  // outGivenExactIn token x for y - polynomial equation to solve                                              //
	  // ay = amount out to calculate                                                                              //
	  // by = balance token out                                                                                    //
	  // y = by - ay (finalBalanceOut)                                                                             //
	  // D = invariant                                               D                     D^(n+1)                 //
	  // A = amplification coefficient               y^2 + ( S + ----------  - D) * y -  ------------- = 0         //
	  // n = number of tokens                                    (A * n^n)               A * n^2n * P              //
	  // S = sum of final balances but y                                                                           //
	  // P = product of final balances but y                                                                       //
	  **************************************************************************************************************/

	s.mu.Lock()
	defer s.mu.Unlock()

	balances[tokenIndexIn], err = FixPoint.Add(balances[tokenIndexIn], tokenAmountIn)
	if err != nil {
		return nil, err
	}

	finalBalanceOut, err := s.ComputeBalance(amplificationParameter, balances, invariant, tokenIndexOut)
	if err != nil {
		return nil, err
	}

	balances[tokenIndexIn], err = FixPoint.Sub(balances[tokenIndexIn], tokenAmountIn)
	if err != nil {
		return nil, err
	}

	amountOut, err = FixPoint.Sub(balances[tokenIndexOut], finalBalanceOut)
	if err != nil {
		return nil, err
	}

	amountOut.SubUint64(amountOut, 1)

	return
}

func (s *stableMath) ComputeInGivenExactOut(
	amplificationParameter *uint256.Int,
	balances []*uint256.Int,
	tokenIndexIn, tokenIndexOut int,
	tokenAmountOut, invariant *uint256.Int,
) (amountIn *uint256.Int, err error) {
	/**************************************************************************************************************
	  // inGivenExactOut token x for y - polynomial equation to solve                                              //
	  // ax = amount in to calculate                                                                               //
	  // bx = balance token in                                                                                     //
	  // x = bx + ax (finalBalanceIn)                                                                              //
	  // D = invariant                                                D                     D^(n+1)                //
	  // A = amplification coefficient               x^2 + ( S + ----------  - D) * x -  ------------- = 0         //
	  // n = number of tokens                                     (A * n^n)               A * n^2n * P             //
	  // S = sum of final balances but x                                                                           //
	  // P = product of final balances but x                                                                       //
	  **************************************************************************************************************/

	s.mu.Lock()
	defer s.mu.Unlock()

	balances[tokenIndexOut], err = FixPoint.Sub(balances[tokenIndexOut], tokenAmountOut)
	if err != nil {
		return nil, err
	}

	finalBalanceIn, err := s.ComputeBalance(amplificationParameter, balances, invariant, tokenIndexIn)
	if err != nil {
		return nil, err
	}

	balances[tokenIndexOut], err = FixPoint.Add(balances[tokenIndexOut], tokenAmountOut)
	if err != nil {
		return nil, err
	}

	amountIn, err = FixPoint.Sub(finalBalanceIn, balances[tokenIndexIn])
	if err != nil {
		return nil, err
	}

	amountIn.AddUint64(amountIn, 1)

	return
}

func (s *stableMath) ComputeBalance(
	amplificationParameter *uint256.Int,
	balances []*uint256.Int,
	invariant *uint256.Int,
	tokenIndex int,
) (*uint256.Int, error) {
	numTokens := uint256.NewInt(uint64(len(balances)))

	// A * n
	ampTimesN := new(uint256.Int).Mul(amplificationParameter, numTokens)

	sumBalances := new(uint256.Int).Set(balances[0])
	balanceProduct := new(uint256.Int).Mul(balances[0], numTokens)

	// (P_D * x_j * n) / D
	mulResult := new(uint256.Int)
	for j := 1; j < len(balances); j++ {
		mulResult.Mul(balanceProduct, balances[j])
		mulResult.Mul(mulResult, numTokens)
		balanceProduct.Div(mulResult, invariant)
		sumBalances.Add(sumBalances, balances[j])
	}

	sumBalances.Sub(sumBalances, balances[tokenIndex])

	invariantSquared := new(uint256.Int).Mul(invariant, invariant)

	// c = (D^2 * AP)/(An * P_D) * x_i
	numerator := new(uint256.Int).Mul(invariantSquared, _AMP_PRECISION)
	denominator := new(uint256.Int).Mul(ampTimesN, balanceProduct)
	c := new(uint256.Int).Div(numerator, denominator)
	c.Mul(c, balances[tokenIndex])

	// b = S + (D * AP)/An
	b := new(uint256.Int).Mul(invariant, _AMP_PRECISION)
	b.Div(b, ampTimesN)
	b.Add(b, sumBalances)

	// y = (D^2 + c)/(D + b)
	numerator.Add(invariantSquared, c)
	denominator.Add(invariant, b)
	tokenBalance := new(uint256.Int).Div(numerator, denominator)

	prevTokenBalance := new(uint256.Int)
	for i := 0; i < 255; i++ {
		prevTokenBalance.Set(tokenBalance)

		// y = (y^2 + c)/(2y + b - D)
		numerator.Mul(tokenBalance, tokenBalance)
		numerator.Add(numerator, c)

		denominator.Mul(tokenBalance, TWO)
		denominator.Add(denominator, b)
		denominator.Sub(denominator, invariant)

		tokenBalance.Div(numerator, denominator)

		if tokenBalance.Gt(prevTokenBalance) {
			mulResult.Sub(tokenBalance, prevTokenBalance)
			if mulResult.Cmp(ONE) <= 0 {
				return tokenBalance, nil
			}
		} else {
			mulResult.Sub(prevTokenBalance, tokenBalance)
			if mulResult.Cmp(ONE) <= 0 {
				return tokenBalance, nil
			}
		}
	}

	return nil, ErrStableComputeBalanceDidNotConverge
}

func (s *stableMath) ComputeInvariant(amplificationParameter *uint256.Int, balances []*uint256.Int) (*uint256.Int, error) {
	/**********************************************************************************************
	  // invariant                                                                                 //
	  // D = invariant                                                  D^(n+1)                    //
	  // A = amplification coefficient      A  n^n S + D = A D n^n + -----------                   //
	  // S = sum of balances                                             n^n P                     //
	  // P = product of balances                                                                   //
	  // n = number of tokens                                                                      //
	  **********************************************************************************************/

	numTokens := uint256.NewInt(uint64(len(balances)))
	sum := uint256.NewInt(0)

	s.mu.RLock()
	for i := range balances {
		sum.Add(sum, balances[i])
	}
	s.mu.RUnlock()

	if sum.IsZero() {
		return sum, nil
	}

	prevInvariant := new(uint256.Int)                                    // Dprev in the Curve version
	invariant := new(uint256.Int).Set(sum)                               // D in the Curve version
	ampTimesN := new(uint256.Int).Mul(amplificationParameter, numTokens) // Ann in the Curve version

	tmp := new(uint256.Int)
	D_P := new(uint256.Int)
	numer := new(uint256.Int)
	denom := new(uint256.Int)
	diff := new(uint256.Int)

	for i := 0; i < 255; i++ {
		prevInvariant.Set(invariant)

		// D_P = D^(n+1)/(n^n * P)
		D_P.Set(invariant)

		s.mu.RLock()
		for _, balance := range balances {
			// D_P = D_P * D / (x_i * n)
			tmp.Mul(balance, numTokens)
			D_P.Mul(D_P, invariant)
			D_P.Div(D_P, tmp)
		}
		s.mu.RUnlock()

		// (A * n * S / AP + D_P * n) * D
		numer.Mul(ampTimesN, sum)
		numer.Div(numer, _AMP_PRECISION)
		tmp.Mul(D_P, numTokens)
		numer.Add(numer, tmp)
		numer.Mul(numer, invariant)

		// ((A * n - AP) * D / AP + (n + 1) * D_P)
		denom.Sub(ampTimesN, _AMP_PRECISION)
		denom.Mul(denom, invariant)
		denom.Div(denom, _AMP_PRECISION)
		tmp.AddUint64(numTokens, 1)
		tmp.Mul(tmp, D_P)
		denom.Add(denom, tmp)

		invariant.Div(numer, denom)

		if invariant.Gt(prevInvariant) {
			diff.Sub(invariant, prevInvariant)
		} else {
			diff.Sub(prevInvariant, invariant)
		}
		if diff.IsUint64() && diff.Uint64() <= 1 {
			return invariant, nil
		}
	}

	return nil, ErrStableInvariantDidNotConverge
}
