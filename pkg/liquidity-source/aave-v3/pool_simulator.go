package aavev3

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	extra Extra

	nextLiquidityIndex *uint256.Int
	poolAddress        string
	gas                Gas
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

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
		extra:       extra,
		poolAddress: extra.PoolAddress,
		gas: Gas{
			Supply:   150000,
			Withdraw: 100000,
		},
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
	)

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, fmt.Errorf("invalid token")
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, fmt.Errorf("invalid amount in")
	}

	if amountIn.Sign() <= 0 {
		return nil, fmt.Errorf("insufficient input amount")
	}

	reserveIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexIn])
	if overflow {
		return nil, fmt.Errorf("invalid reserve")
	}

	if reserveIn.Sign() <= 0 {
		return nil, fmt.Errorf("insufficient liquidity")
	}

	// Aave V3 operations based on Solidity logic:
	// 1. Supply: deposit asset to get aToken (1:1 ratio)
	// 2. Withdraw: burn aToken to get underlying asset (1:1 ratio)

	// Supply operation: asset -> aToken
	if s.isSupplyOperation(tokenAmountIn.Token, tokenOut) {
		// Validate supply cap if exists
		if s.extra.SupplyCap != nil && s.extra.SupplyCap.Sign() > 0 {
			currentSupply := new(uint256.Int).Set(reserveIn)
			newSupply := new(uint256.Int).Add(currentSupply, amountIn)
			supplyCap, _ := uint256.FromBig(s.extra.SupplyCap)
			if newSupply.Cmp(supplyCap) > 0 {
				return nil, ErrSupplyCapExceeded
			}
		}

		// Supply: use liquidity index to calculate aToken amount
		// From AToken._mintScaled(): amountScaled = amount.rayDiv(index)
		amountOut, err := s.mint(amountIn)
		if err != nil {
			return nil, fmt.Errorf("mint failed: %w", err)
		}
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
			Fee:            &pool.TokenAmount{Token: tokenAmountIn.Token, Amount: integer.Zero()},
			Gas:            s.gas.Supply,
		}, nil
	}

	// Withdraw operation: aToken -> asset
	if s.isWithdrawOperation(tokenAmountIn.Token, tokenOut) {
		// Withdraw: use liquidity index to calculate underlying amount
		// From AToken._burnScaled(): amountOut = amountIn.rayMul(index)
		amountOut, err := s.burn(amountIn)
		if err != nil {
			return nil, fmt.Errorf("burn failed: %w", err)
		}
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
			Fee:            &pool.TokenAmount{Token: tokenAmountIn.Token, Amount: integer.Zero()},
			Gas:            s.gas.Withdraw,
		}, nil
	}

	return nil, fmt.Errorf("unsupported operation")
}

func (s *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	var (
		tokenAmountOut = param.TokenAmountOut
		tokenIn        = param.TokenIn
	)

	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, fmt.Errorf("invalid token")
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, fmt.Errorf("invalid amount out")
	}

	if amountOut.Sign() <= 0 {
		return nil, fmt.Errorf("insufficient output amount")
	}

	reserveOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexOut])
	if overflow {
		return nil, fmt.Errorf("invalid reserve")
	}

	if reserveOut.Sign() <= 0 {
		return nil, fmt.Errorf("insufficient liquidity")
	}

	// Supply: calculate amount of asset needed to get specific amount of aToken
	if s.isSupplyOperation(tokenIn, tokenAmountOut.Token) {
		// Supply: amountIn = amountOut.rayMul(index) (reverse of mint)
		amountIn, err := s.burn(amountOut)
		if err != nil {
			return nil, ErrMintFailed
		}
		return &pool.CalcAmountInResult{
			TokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: amountIn.ToBig()},
			Fee:           &pool.TokenAmount{Token: tokenIn, Amount: bignumber.ZeroBI},
			Gas:           s.gas.Supply,
		}, nil
	}

	// Withdraw: calculate amount of aToken needed to get specific amount of underlying
	if s.isWithdrawOperation(tokenIn, tokenAmountOut.Token) {
		// Withdraw: amountIn = amountOut.rayDiv(index) (reverse of burn)
		amountIn, err := s.mint(amountOut)
		if err != nil {
			return nil, ErrBurnFailed
		}
		return &pool.CalcAmountInResult{
			TokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: amountIn.ToBig()},
			Fee:           &pool.TokenAmount{Token: tokenIn, Amount: bignumber.ZeroBI},
			Gas:           s.gas.Withdraw,
		}, nil
	}

	return nil, fmt.Errorf("unsupported operation")
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	// Based on Aave V3 Solidity logic:
	// Supply: decrease input token (asset), increase output token (aToken)
	if s.isSupplyOperation(params.TokenAmountIn.Token, params.TokenAmountOut.Token) {
		s.Pool.Info.Reserves[indexIn] = new(big.Int).Sub(s.Pool.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
		s.Pool.Info.Reserves[indexOut] = new(big.Int).Add(s.Pool.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
	} else if s.isWithdrawOperation(params.TokenAmountIn.Token, params.TokenAmountOut.Token) {
		// Withdraw: decrease input token (aToken), increase output token (asset)
		s.Pool.Info.Reserves[indexIn] = new(big.Int).Sub(s.Pool.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
		s.Pool.Info.Reserves[indexOut] = new(big.Int).Add(s.Pool.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
	}
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		Extra: s.extra,
		PoolMetaGeneric: PoolMetaGeneric{
			ApprovalAddress: s.poolAddress,
			NoFOT:           true,
		},
	}
}

func (s *PoolSimulator) isSupplyOperation(tokenIn, tokenOut string) bool {
	// Supply: deposit underlying asset to get aToken
	// tokenIn = underlying asset, tokenOut = aToken
	return tokenIn != tokenOut && s.extra.ATokenAddress == tokenOut
}

func (s *PoolSimulator) isWithdrawOperation(tokenIn, tokenOut string) bool {
	// Withdraw: burn aToken to get underlying asset
	// tokenIn = aToken, tokenOut = underlying asset
	return tokenIn != tokenOut && s.extra.ATokenAddress == tokenIn
}

func (s *PoolSimulator) mint(amount *uint256.Int) (*uint256.Int, error) {
	amountScaled, err := rayDiv(amount, s.nextLiquidityIndex)
	if err != nil {
		return nil, err
	}

	if amountScaled.IsZero() {
		return nil, ErrInvalidMintAmount
	}

	return amountScaled, nil
}

func (s *PoolSimulator) burn(amountIn *uint256.Int) (*uint256.Int, error) {
	liquidityIndex := s.extra.LiquidityIndex
	if liquidityIndex == nil {
		amountOut, err := rayMul(amountIn, RAY)
		if err != nil {
			return nil, ErrBurnFailed
		}
		return amountOut, nil
	}

	var liquidityIndexU256 uint256.Int
	liquidityIndexU256.SetFromBig(liquidityIndex)
	amountOut, err := rayMul(amountIn, &liquidityIndexU256)
	if err != nil {
		return nil, ErrBurnFailed
	}

	return amountOut, nil
}
