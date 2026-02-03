package stabull

import (
	"encoding/json"
	"errors"
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
	decs, reserves [2]*uint256.Int
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
			Reserves: []*big.Int{bignumber.NewBig10(ep.Reserves[0]), bignumber.NewBig10(ep.Reserves[1])},
		}},
		reserves: [2]*uint256.Int{big256.New(ep.Reserves[0]), big256.New(ep.Reserves[1])},
		decs:     [2]*uint256.Int{big256.TenPow(ep.Tokens[0].Decimals), big256.TenPow(ep.Tokens[1].Decimals)},
		Extra:    extra,
	}, nil
}

// CalcAmountOut calculates the expected output amount for a given input
// Uses cached reserve and curve parameter state from pool tracker
//
// implements the Stabull curve swap calculation logic
// Stabull uses a sophisticated invariant-based curve with oracle integration
// The actual contract uses viewOriginSwap(origin, target, amount) which implements:
// 1. Hybrid constant product and constant sum invariant
// 2. Dynamic pricing based on pool balance vs oracle rate
// 3. Curve parameters (alpha, beta, delta, epsilon, lambda) define the shape
// 4. Dynamic fee based on epsilon and pool imbalance
//
// We implement the curve math using the greek parameters from pool state
func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmount
	}

	reserveIn, reserveOut := s.reserves[indexIn], s.reserves[indexOut]
	if reserveIn == nil || reserveIn.Sign() <= 0 {
		return nil, errors.New("insufficient reserve in")
	} else if reserveOut == nil || reserveOut.Sign() <= 0 {
		return nil, errors.New("insufficient reserve out")
	}

	inputOracleRate, outputOracleRate := s.OracleRates[indexIn], s.OracleRates[indexOut]
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
	amtInNumeraire, _ := amountIn.MulDivOverflow(amountIn, inputOracleRate, OracleDecimals)
	amtInNumeraire = divu(amtInNumeraire, s.decs[indexIn])

	var tmp1, tmp2 uint256.Int
	tmp1.MulDivOverflow(reserveIn, inputOracleRate, OracleDecimals)
	resInNumeraire := divu(&tmp1, s.decs[indexIn])

	tmp2.MulDivOverflow(reserveOut, outputOracleRate, OracleDecimals)
	resOutNumeraire := divu(&tmp2, s.decs[indexOut])

	// Use the Stabull curve formula with greek parameters
	amtOutNumeraire, err := calculateTrade(amtInNumeraire, resInNumeraire, resOutNumeraire, s.Alpha, s.Beta, s.Delta,
		s.Lambda)
	if err != nil {
		return nil, err
	}
	// Apply epsilon fee: result = result * (ONE - epsilon) / ONE
	// In the contract: _amt = _amt.us_mul(ONE - curve.epsilon)
	fee := tmp1.Sub(big256.U2Pow64, s.Epsilon)
	amtOutNumeraire = usMul(amtOutNumeraire, fee)

	amountOut := mulu(amtOutNumeraire, s.decs[indexOut])
	// Convert output from numeraire to token decimals: (amountOutNumeraire * 1e8) / outputOracleRate
	amountOut.MulDivOverflow(amountOut, OracleDecimals, outputOracleRate)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenAmountIn.Token, Amount: bignumber.ZeroBI},
		Gas:            defaultGas,
	}, nil
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
