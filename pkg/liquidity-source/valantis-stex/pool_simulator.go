package valantisstex

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	Extra
	StaticExtra
	reserve0, reserve1 *uint256.Int
}

type SwapInfo struct {
	IsZeroToOne bool `json:"isZeroToOne"`
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves:    lo.Map(ep.Reserves, func(item string, _ int) *big.Int { return bignum.NewBig(item) }),
			BlockNumber: ep.BlockNumber,
		}},
		Extra:       extra,
		StaticExtra: staticExtra,
		reserve0:    uint256.MustFromDecimal(ep.Reserves[0]),
		reserve1:    uint256.MustFromDecimal(ep.Reserves[1]),
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut
	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	isZeroToOne := indexIn == 0

	feeInBips := s.DefaultSwapFeeBips
	if !valueobject.IsZeroAddress(s.SwapFeeModule) {
		feeInBips = lo.Ternary(isZeroToOne, s.SwapFeeInBipsZtoO, s.SwapFeeInBipsOtoZ).Clone()
		if feeInBips.Gt(maxSwapFeeBips) {
			return nil, ErrSovereignPoolSwapExcessiveSwapFee
		}
	}

	amountIn := uint256.MustFromBig(tokenAmountIn.Amount)
	if amountIn.IsZero() {
		return nil, ErrZeroSwap
	}

	amountInWithoutFee, overflow := new(uint256.Int).MulDivOverflow(
		amountIn, maxSwapFeeBips,
		new(uint256.Int).Add(maxSwapFeeBips, feeInBips),
	)
	if overflow {
		return nil, number.ErrOverflow
	}

	var amountOut *uint256.Int
	var err error
	if isZeroToOne {
		amountOut, err = s.convertToToken1(amountInWithoutFee)
	} else {
		amountOut, err = s.convertToToken0(amountInWithoutFee)
	}
	if err != nil {
		return nil, err
	}
	if amountOut.Gt(lo.Ternary(isZeroToOne, s.reserve1, s.reserve0)) {
		return nil, ErrInsufficientReserve
	}

	if s.Gas[0] == 0 || s.Gas[1] == 0 {
		return nil, ErrInvalidGasConfig
	}

	fee := new(uint256.Int).Sub(amountIn, amountInWithoutFee)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: fee.ToBig(),
		},
		Gas: int64(lo.Ternary(isZeroToOne, s.Gas[0], s.Gas[1])),
	}, nil
}

func (s *PoolSimulator) convertToToken0(amount *uint256.Int) (*uint256.Int, error) {
	res, overflow := new(uint256.Int).MulDivOverflow(amount, s.Rate1To0, u256.BONE)
	if overflow {
		return nil, number.ErrOverflow
	}

	return res, nil
}

func (s *PoolSimulator) convertToToken1(amount *uint256.Int) (*uint256.Int, error) {
	res, overflow := new(uint256.Int).MulDivOverflow(amount, s.Rate0To1, u256.BONE)
	if overflow {
		return nil, number.ErrOverflow
	}

	return res, nil
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	return MetaInfo{
		BlockNumber: s.Info.BlockNumber,
		IsZeroToOne: s.GetTokenIndex(tokenIn) == 0,
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	s.reserve0 = new(uint256.Int).Set(s.reserve0)
	s.reserve1 = new(uint256.Int).Set(s.reserve1)

	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn := s.GetTokenIndex(params.TokenAmountIn.Token)
	isZeroToOne := indexIn == 0

	if isZeroToOne {
		s.reserve0.Add(s.reserve0, uint256.MustFromBig(params.TokenAmountIn.Amount))
		s.reserve1.Sub(s.reserve1, uint256.MustFromBig(params.TokenAmountOut.Amount))
	} else {
		s.reserve0.Sub(s.reserve0, uint256.MustFromBig(params.TokenAmountOut.Amount))
		s.reserve1.Add(s.reserve1, uint256.MustFromBig(params.TokenAmountIn.Amount))
	}
}
