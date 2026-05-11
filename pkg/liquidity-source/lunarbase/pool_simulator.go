package lunarbase

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	reserves []*uint256.Int
	chainID  valueobject.ChainID
	*Extra
	*StaticExtra
}

type SwapInfo struct {
	nextPX96 *uint256.Int
}

var _ = pool.RegisterFactory(DexType, NewPoolSimulator)

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	entityPool, chainID := params.EntityPool, params.ChainID
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	} else if params.Opts.StaleCheck && extra.IsStale(entityPool.BlockNumber) {
		return nil, ErrStalePool
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  entityPool.Address,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens: lo.Map(entityPool.Tokens,
					func(item *entity.PoolToken, _ int) string { return item.Address }),
				Reserves: lo.Map(entityPool.Reserves,
					func(item string, _ int) *big.Int { return bignumber.NewBig(item) }),
				BlockNumber: entityPool.BlockNumber,
			},
		},
		chainID: chainID,
		reserves: lo.Map(entityPool.Reserves, func(item string, _ int) *uint256.Int {
			return big256.New(item)
		}),
		Extra:       &extra,
		StaticExtra: &staticExtra,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	} else if s.Paused {
		return nil, ErrPoolPaused
	} else if s.PriceX96 == nil || s.PriceX96.IsZero() {
		return nil, ErrZeroPrice
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow || amountIn.Sign() <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	poolParams := &PoolParams{
		SqrtPriceX96:   s.PriceX96,
		FeeQ48:         s.FeeQ48,
		ReserveX:       s.reserves[0],
		ReserveY:       s.reserves[1],
		ConcentrationK: s.ConcentrationK,
	}

	var result *QuoteResult
	if indexIn == 0 {
		result = quoteXToY(poolParams, amountIn)
	} else {
		result = quoteYToX(poolParams, amountIn)
	}

	if result.AmountOut.IsZero() {
		return nil, ErrInsufficientLiquidity
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: result.AmountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenOut, Amount: result.Fee.ToBig()},
		Gas:            defaultGas,
		SwapInfo:       SwapInfo{nextPX96: result.SqrtPriceNext},
	}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Extra = lo.ToPtr(*s.Extra)
	cloned.StaticExtra = lo.ToPtr(*s.StaticExtra)
	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	inAmount := uint256.MustFromBig(params.TokenAmountIn.Amount)
	outAmount := uint256.MustFromBig(params.TokenAmountOut.Amount)
	s.reserves = slices.Clone(s.reserves)
	s.reserves[indexIn] = inAmount.Add(s.reserves[indexIn], inAmount)
	s.reserves[indexOut] = outAmount.Sub(s.reserves[indexOut], outAmount)
	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok && swapInfo.nextPX96 != nil {
		s.PriceX96 = swapInfo.nextPX96
	}
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return PoolMeta{
		BlockNumber:     s.Info.BlockNumber,
		ApprovalAddress: s.GetApprovalAddress(tokenIn, tokenOut),
		HasNative:       s.HasNative,
	}
}

func (s *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	if !s.HasNative || !valueobject.IsWrappedNative(tokenIn, s.chainID) {
		return s.GetAddress()
	}
	return ""
}

func (s *PoolSimulator) SwapReceiveNativeIn(tokenIn, tokenOut string, _ valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(PoolMeta)
	return meta.HasNative && valueobject.IsWrappedNative(tokenIn, s.chainID)
}

func (s *PoolSimulator) SwapReturnNativeOut(tokenIn, tokenOut string, _ valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(PoolMeta)
	return meta.HasNative && valueobject.IsWrappedNative(tokenOut, s.chainID)
}
