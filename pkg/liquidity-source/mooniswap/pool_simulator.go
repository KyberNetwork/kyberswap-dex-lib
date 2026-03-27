package mooniswap

import (
	"math/big"
	"slices"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	fee         *uint256.Int
	slippageFee *uint256.Int
	balAdd      [2]*uint256.Int
	balRem      [2]*uint256.Int

	StaticExtra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, _ int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		fee:         extra.Fee,
		slippageFee: extra.SlippageFee,
		balAdd:      [2]*uint256.Int{extra.BalAdd0, extra.BalAdd1},
		balRem:      [2]*uint256.Int{extra.BalRem0, extra.BalRem1},
		StaticExtra: staticExtra,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn := s.GetTokenIndex(param.TokenAmountIn.Token)
	indexOut := s.GetTokenIndex(param.TokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(param.TokenAmountIn.Amount)
	if overflow || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	srcBalance := s.balAdd[indexIn]
	dstBalance := s.balRem[indexOut]

	amountOut := calcAmountOut(amountIn, srcBalance, dstBalance, s.fee, s.slippageFee)
	if amountOut.IsZero() {
		return nil, ErrZeroAmount
	}

	reserveOut, _ := uint256.FromBig(s.Info.Reserves[indexOut])
	if amountOut.Gt(reserveOut) {
		return nil, ErrInsufficientLiquidity
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[indexIn], Amount: bignumber.ZeroBI},
		Gas:            defaultGas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn := s.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	amtIn := uint256.MustFromBig(params.TokenAmountIn.Amount)
	amtOut := uint256.MustFromBig(params.TokenAmountOut.Amount)

	s.balAdd[indexIn] = new(uint256.Int).Add(s.balAdd[indexIn], amtIn)
	s.balRem[indexOut] = new(uint256.Int).Sub(s.balRem[indexOut], amtOut)
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.balAdd = [2]*uint256.Int{
		new(uint256.Int).Set(s.balAdd[0]),
		new(uint256.Int).Set(s.balAdd[1]),
	}
	cloned.balRem = [2]*uint256.Int{
		new(uint256.Int).Set(s.balRem[0]),
		new(uint256.Int).Set(s.balRem[1]),
	}
	cloned.Info.Reserves = slices.Clone(s.Info.Reserves)

	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	tokenInIndex := s.GetTokenIndex(tokenIn)
	return PoolMeta{
		IsNativeIn:  lo.Ternary(tokenInIndex == 0, s.IsNativeToken0, s.IsNativeToken1),
		IsNativeOut: lo.Ternary(tokenInIndex == 0, s.IsNativeToken1, s.IsNativeToken0),
	}
}
