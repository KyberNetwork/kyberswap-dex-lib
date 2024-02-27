package bancor_v21

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type (
	PoolSimulator struct {
		poolpkg.Pool
		fee *uint256.Int
		gas Gas
	}
)

var (
	ErrInvalidReserve = errors.New("invalid reserve")
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
			Address:     entityPool.Address,
			ReserveUsd:  entityPool.ReserveUsd,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return utils.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		fee: uint256.NewInt(extra.conversionFee),
		gas: defaultGas,
	}, nil
}

// crossReserveTargetAmount calculates the target amount in a cross-reserve operation
// using big integers for arbitrary precision arithmetic.
// reference: https://github.com/bancorprotocol/contracts-solidity/blob/dc378ab9d57d1b4a41dfa95fc5142fac2f4ee307/contracts/converter/types/standard-pool/StandardPoolConverter.sol#L1082
func crossReserveTargetAmount(sourceReserveBalance, targetReserveBalance, sourceAmount *big.Int) (*big.Int, error) {
	// Ensure that both source and target reserve balances are greater than 0
	if sourceReserveBalance.Cmp(utils.ZeroBI) != 1 || targetReserveBalance.Cmp(utils.ZeroBI) != 1 {
		return nil, ErrInvalidReserve
	}

	// Perform the calculation: targetReserveBalance * sourceAmount / (sourceReserveBalance + sourceAmount)
	numerator := new(big.Int).Mul(targetReserveBalance, sourceAmount)
	denominator := new(big.Int).Add(sourceReserveBalance, sourceAmount)
	result := new(big.Int).Div(numerator, denominator)

	return result, nil
}

// getAmountOut calculates the amount of tokenOut to receive for a given amount of tokenIn
// ref: https://github.com/bancorprotocol/contracts-solidity/blob/dc378ab9d57d1b4a41dfa95fc5142fac2f4ee307/contracts/converter/types/standard-pool/StandardPoolConverter.sol#L450
func calculateFee(targetAmount *big.Int, conversionFee *big.Int) *big.Int {
	// Assuming PPM_RESOLUTION is a constant

	// Convert PPM_RESOLUTION to big.Int
	ppmResolution := big.NewInt(PPM_RESOLUTION)

	// Calculate targetAmount * conversionFee
	numerator := new(big.Int).Mul(targetAmount, conversionFee)

	// Calculate the fee: (targetAmount * conversionFee) / PPM_RESOLUTION
	fee := new(big.Int).Div(numerator, ppmResolution)

	return fee
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
	)
	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	if amountIn.Cmp(number.Zero) <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	reserveIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexIn])
	if overflow {
		return nil, ErrInvalidReserve
	}

	reserveOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexOut])
	if overflow {
		return nil, ErrInvalidReserve
	}

	if reserveIn.Cmp(number.Zero) <= 0 || reserveOut.Cmp(number.Zero) <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountOut := s.getAmountOut(amountIn, reserveIn, reserveOut)
	if amountOut.Cmp(reserveOut) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	// NOTE: Intentionally comment out, since kAfter should always smaller than kBefore.
	// balanceIn := new(uint256.Int).Add(reserveIn, amountIn)
	// balanceOut := new(uint256.Int).Sub(reserveOut, amountOut)

	// balanceInAdjusted := new(uint256.Int).Sub(
	// 	new(uint256.Int).Mul(balanceIn, s.feePrecision),
	// 	new(uint256.Int).Mul(amountIn, s.fee),
	// )
	// balanceOutAdjusted := new(uint256.Int).Mul(balanceOut, s.feePrecision)

	// kBefore := new(uint256.Int).Mul(new(uint256.Int).Mul(reserveIn, reserveOut), new(uint256.Int).Mul(s.feePrecision, s.feePrecision))
	// kAfter := new(uint256.Int).Mul(balanceInAdjusted, balanceOutAdjusted)

	// if kAfter.Cmp(kBefore) < 0 {
	// 	return nil, ErrInvalidK
	// }

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		// NOTE: we don't use fee to update balance so that we don't need to calculate it. I put it number.Zero to avoid null pointer exception
		Fee: &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas: s.gas.Swap,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	s.Pool.Info.Reserves[indexIn] = new(big.Int).Add(s.Pool.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	s.Pool.Info.Reserves[indexOut] = new(big.Int).Sub(s.Pool.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		Fee:         s.fee.Uint64(),
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}
