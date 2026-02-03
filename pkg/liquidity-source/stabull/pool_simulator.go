package stabull

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	reserves, powDecs [2]*uint256.Int
	Extra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  strings.ToLower(ep.Address),
			Exchange: ep.Exchange,
			Type:     ep.Type,
			Tokens:   []string{ep.Tokens[0].Address, ep.Tokens[1].Address},
			Reserves: []*big.Int{bignumber.NewBig10(ep.Reserves[0]),
				bignumber.NewBig10(ep.Reserves[1])},
		}},
		reserves: [2]*uint256.Int{big256.New(ep.Reserves[0]), big256.New(ep.Reserves[1])},
		powDecs:  [2]*uint256.Int{big256.TenPow(ep.Tokens[0].Decimals), big256.TenPow(ep.Tokens[1].Decimals)},
		Extra:    extra,
	}, nil
}

// CalcAmountOut calculates the expected output amount for a given input
// Uses cached reserve and curve parameter state from pool tracker
// Expects tokenAmountIn.Amount in input token decimals, returns output in output token decimals
func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, fmt.Errorf("indexIn: %v or indexOut: %v is not correct", indexIn, indexOut)
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmount
	}
	// Calculate swap using Stabull curve formula
	// Note: The actual contract has viewOriginSwap(origin, target, originAmount) that returns targetAmount
	// In the simulator, we need to replicate this logic locally using cached curve parameters
	amountOut, err := s.calculateSwap(amountIn, indexIn, indexOut)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenAmountIn.Token, Amount: bignumber.ZeroBI},
		Gas:            defaultGas,
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
func (s *PoolSimulator) calculateSwap(amountIn *uint256.Int, indexIn, indexOut int) (*uint256.Int, error) {
	if amountIn == nil || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmount
	}

	// Get token decimals from entity pool tokens
	// Reserves are stored in 18 decimals (numeraire), but input/output are in token decimals
	reserveIn, reserveOut := s.reserves[indexIn], s.reserves[indexOut]
	if reserveIn == nil || reserveIn.Sign() <= 0 {
		return nil, errors.New("insufficient reserve in")
	} else if reserveOut == nil || reserveOut.Sign() <= 0 {
		return nil, errors.New("insufficient reserve out")
	}

	inputOracleRate, outputOracleRate := s.OracleRates[indexIn], s.OracleRates[indexOut]

	// Validate oracle rates
	if inputOracleRate == nil || inputOracleRate.Sign() <= 0 {
		if indexIn == 0 {
			return nil, errors.New("missing or invalid BaseOracleRate for input token")
		}
		return nil, errors.New("missing or invalid QuoteOracleRate for input token")
	} else if outputOracleRate == nil || outputOracleRate.Sign() <= 0 {
		if indexOut == 0 {
			return nil, errors.New("missing or invalid BaseOracleRate for output token")
		}
		return nil, errors.New("missing or invalid QuoteOracleRate for output token")
	}

	// Convert input to numeraire: (amountIn * inputOracleRate) / 1e8
	amountInNumeraire, _ := amountIn.MulDivOverflow(amountIn, inputOracleRate, NumerairePrecision)
	// in contract, the number is then divu-ed by 10**tokenDecimals into Q64.64 format, here we just mul by 2**64
	amountInNumeraire.MulDivOverflow(amountInNumeraire, big256.U2Pow64, s.powDecs[indexIn])

	// Use the Stabull curve formula with greek parameters
	amountOutNumeraire, err := calculateStabullSwap(
		amountInNumeraire,
		reserveIn,
		reserveOut,
		s.Alpha,
		s.Beta,
		s.Delta,
		s.Epsilon,
		s.Lambda,
	)
	if err != nil {
		return nil, err
	}

	amountOutNumeraire.MulDivOverflow(amountOutNumeraire, s.powDecs[indexOut], big256.U2Pow64)
	// Convert output from numeraire to token decimals: (amountOutNumeraire * 1e8) / outputOracleRate
	result, _ := amountOutNumeraire.MulDivOverflow(amountOutNumeraire, NumerairePrecision, outputOracleRate)
	return result, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	amtIn, amtOut := uint256.MustFromBig(params.TokenAmountIn.Amount), uint256.MustFromBig(params.TokenAmountOut.Amount)
	s.reserves[indexIn] = new(uint256.Int).Add(s.reserves[indexIn], amtIn)
	s.reserves[indexOut] = new(uint256.Int).Sub(s.reserves[indexOut], amtOut)
}

// GetMetaInfo returns metadata about the pool
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return nil
}
