package lunarbase

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	reserves []*uint256.Int
	chainID  valueobject.ChainID
	// Old serialized K=0/max=0 frames cannot identify the pricing model.
	// Keep the rest of a shared cache readable, but require this pool's refresh.
	requiresRPCRefresh bool
	*Extra
	*StaticExtra
}

type SwapInfo struct {
	nextSqrtPriceX96 *uint256.Int
	nextReserves     [2]uint256.Int
	valid            bool
	// feeAskX24/feeBidX24 are the post-swap persisted directional fees.
	// Unchanged from the pre-swap values on the concentration-curve model;
	// ratcheted up on the swapped direction for a punishment-model pool
	// (SDK v0.4.0), since punishment accumulates per swap.
	feeAskX24 uint32
	feeBidX24 uint32
}

var _ = pool.RegisterFactory(DexType, NewPoolSimulator)

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	entityPool, chainID := params.EntityPool, params.ChainID
	if len(entityPool.Tokens) != 2 || len(entityPool.Reserves) != 2 ||
		entityPool.Tokens[0] == nil || entityPool.Tokens[1] == nil ||
		entityPool.Tokens[0].Address == "" || entityPool.Tokens[1].Address == "" ||
		entityPool.Tokens[0].Address == entityPool.Tokens[1].Address {
		return nil, ErrInvalidToken
	}
	reserves := make([]*uint256.Int, 2)
	reserveInfo := make([]*big.Int, 2)
	for i, raw := range entityPool.Reserves {
		value, ok := new(big.Int).SetString(raw, 10)
		if !ok || value.Sign() < 0 || value.BitLen() > 112 {
			return nil, ErrInsufficientLiquidity
		}
		reserveInfo[i] = value
		reserves[i] = uint256.MustFromBig(value)
	}
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
				Reserves:    reserveInfo,
				BlockNumber: entityPool.BlockNumber,
			},
		},
		chainID:            chainID,
		requiresRPCRefresh: !extra.ConcentrationModel && extra.ConcentrationK == 0 && extra.MaxPunishmentX24 == 0 && extra.BlockHash == "",
		reserves:           reserves,
		Extra:              &extra,
		StaticExtra:        &staticExtra,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if s.requiresRPCRefresh {
		return nil, ErrStalePool
	}
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	} else if s.IsStale(s.Info.BlockNumber) {
		return nil, ErrStalePool
	} else if s.Paused {
		return nil, ErrPoolPaused
	} else if s.SqrtPriceX96 == nil || s.SqrtPriceX96.IsZero() {
		return nil, ErrZeroPrice
	}

	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrInsufficientLiquidity
	}
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow || amountIn.Sign() <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	poolParams := &PoolParams{
		SqrtPriceX96:       s.SqrtPriceX96,
		FeeAskX24:          s.FeeAskX24,
		FeeBidX24:          s.FeeBidX24,
		ReserveX:           s.reserves[0],
		ReserveY:           s.reserves[1],
		ConcentrationK:     s.ConcentrationK,
		ConcentrationModel: s.ConcentrationModel,
		MaxPunishmentX24:   s.MaxPunishmentX24,
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

	// Sync records net reserves: all output fees leave the tradable reserve
	// through the partner and treasury buckets. Carry the complete post-state
	// into UpdateBalance so it cannot depend on mutable caller-supplied fees.
	var nextReserves [2]uint256.Int
	if _, overflow := nextReserves[indexIn].AddOverflow(s.reserves[indexIn], amountIn); overflow || nextReserves[indexIn].BitLen() > 112 {
		return nil, ErrInsufficientLiquidity
	}
	var grossOut uint256.Int
	if _, overflow := grossOut.AddOverflow(result.AmountOut, result.Fee); overflow || grossOut.Gt(s.reserves[indexOut]) {
		return nil, ErrInsufficientLiquidity
	}
	nextReserves[indexOut].Sub(s.reserves[indexOut], &grossOut)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: result.AmountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenOut, Amount: result.Fee.ToBig()},
		Gas:            defaultGas,
		SwapInfo: SwapInfo{
			nextSqrtPriceX96: result.SqrtPriceNext,
			nextReserves:     nextReserves,
			valid:            true,
			// quoteXToY/quoteYToX mutate poolParams' fees in place for a
			// punishment-model pool (no-op for the concentration-curve model).
			feeAskX24: poolParams.FeeAskX24,
			feeBidX24: poolParams.FeeBidX24,
		},
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
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return
	}
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok || !swapInfo.valid {
		return
	}
	s.reserves = []*uint256.Int{new(uint256.Int).Set(&swapInfo.nextReserves[0]), new(uint256.Int).Set(&swapInfo.nextReserves[1])}
	s.Info.Reserves = []*big.Int{s.reserves[0].ToBig(), s.reserves[1].ToBig()}
	// SqrtPriceX96 is operator-set on the fix/incident contract; swaps do not
	// mutate it. The SwapInfo.nextSqrtPriceX96 field is kept for diagnostic
	// continuity but intentionally not written back to state.

	// A punishment-model pool ratchets up the swapped direction's fee on
	// every swap; persist it so subsequent hops/splits through the same
	// pool within one route price against the post-swap fee.
	s.Extra = lo.ToPtr(*s.Extra)
	s.FeeAskX24 = swapInfo.feeAskX24
	s.FeeBidX24 = swapInfo.feeBidX24
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return PoolMeta{
		BlockNumber:     s.Info.BlockNumber,
		BlockHash:       s.BlockHash,
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
