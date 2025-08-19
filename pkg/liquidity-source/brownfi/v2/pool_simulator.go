package brownfiv2

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	decimals [2]uint8
	reserves [2]*uint256.Int
	limits   [2]*uint256.Int
	oPrices  [2]*uint256.Int
	fee      *uint256.Int
	lambda   *uint256.Int
	kappa    *uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	if len(entityPool.Tokens) != 2 {
		return nil, ErrInvalidToken
	} else if len(entityPool.Reserves) != 2 {
		return nil, ErrInvalidReserve
	} else if extra.OPrices[0] == nil || extra.OPrices[1] == nil {
		return nil, ErrInvalidPrices
	}
	tokens := make([]string, 2)
	reserves := make([]*big.Int, 2)
	var uReserves [2]*uint256.Int
	var limits [2]*uint256.Int
	var decimals [2]uint8
	for i, token := range entityPool.Tokens {
		reserve, err := uint256.FromDecimal(entityPool.Reserves[i])
		if err != nil {
			return nil, ErrInvalidReserve
		}
		reserves[i] = reserve.ToBig()
		uReserves[i] = reserve
		limits[i] = parseRawToDefaultDecimals(reserve, token.Decimals)
		limits[i].MulDivOverflow(limits[i], big256.U8, big256.U10)
		tokens[i] = token.Address
		decimals[i] = token.Decimals
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: entityPool.BlockNumber,
		}},
		decimals: decimals,
		reserves: uReserves,
		limits:   limits,
		fee:      uint256.NewInt(extra.Fee),
		lambda:   uint256.NewInt(extra.Lambda),
		kappa:    extra.Kappa,
		oPrices:  extra.OPrices,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	parsedAmountIn := parseRawToDefaultDecimals(amountIn, s.decimals[indexIn])
	parsedReserveIn := parseRawToDefaultDecimals(s.reserves[indexIn], s.decimals[indexIn])
	parsedReserveOut := parseRawToDefaultDecimals(s.reserves[indexOut], s.decimals[indexOut])
	priceIn, priceOut := s.oPrices[indexIn], s.oPrices[indexOut]
	priceIn, priceOut = getSkewnessPrice(parsedReserveIn, parsedReserveOut, priceIn, priceOut, s.lambda)

	amountOut := getAmountOut(parsedAmountIn, parsedReserveOut, priceIn, priceOut, s.kappa, s.fee)
	if amountOut.Cmp(s.limits[indexOut]) > 0 {
		return nil, ErrMax80PercentOfReserve
	} else if amountOut.Sign() <= 0 {
		return nil, ErrInsufficientOutputAmount
	}

	amountOut = parseDefaultToRawDecimals(amountOut, s.decimals[indexOut])

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[indexIn], Amount: bignumber.ZeroBI},
		Gas:            defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenAmountOut, tokenIn := param.TokenAmountOut, param.TokenIn
	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow || amountOut.Sign() <= 0 {
		return nil, ErrInvalidAmountOut
	} else if amountOut.Cmp(s.limits[indexOut]) > 0 {
		return nil, ErrMax80PercentOfReserve
	}

	parsedAmountOut := parseRawToDefaultDecimals(amountOut, s.decimals[indexOut])
	parsedReserveIn := parseRawToDefaultDecimals(s.reserves[indexIn], s.decimals[indexIn])
	parsedReserveOut := parseRawToDefaultDecimals(s.reserves[indexOut], s.decimals[indexOut])
	priceIn, priceOut := s.oPrices[indexIn], s.oPrices[indexOut]
	priceIn, priceOut = getSkewnessPrice(parsedReserveIn, parsedReserveOut, priceIn, priceOut, s.lambda)

	amountIn := getAmountIn(parsedAmountOut, parsedReserveOut, priceIn, priceOut, s.kappa, s.fee)
	if amountIn.Sign() <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	amountIn = parseDefaultToRawDecimals(amountIn, s.decimals[indexIn])

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: s.Info.Tokens[indexIn], Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: s.Info.Tokens[indexOut], Amount: bignumber.ZeroBI},
		Gas:           defaultGas,
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
	s.limits[indexOut] = new(uint256.Int).Sub(s.limits[indexOut], amtOut)
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		Fee: s.fee.Uint64(),
	}
}
