package lunarbase

import (
	"math/big"
	"strings"

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

	chainID        valueobject.ChainID
	periphery      string
	permit2        string
	wrappedNative  string
	rawTokenX      string
	rawTokenY      string
	priceX96       *uint256.Int
	feeQ48         uint64
	latestBlock    uint64
	concentrationK uint32
	paused         bool
	reserves       []*uint256.Int
	gas            int64
}

type SwapInfo struct {
	NextPX96 *uint256.Int
}

var _ = pool.RegisterFactory(DexType, NewPoolSimulatorFactory)

func NewPoolSimulatorFactory(params pool.FactoryParams) (*PoolSimulator, error) {
	return newPoolSimulator(params.EntityPool, params.ChainID)
}

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	return newPoolSimulator(entityPool, chainID)
}

func newPoolSimulator(
	entityPool entity.Pool,
	chainID valueobject.ChainID,
) (*PoolSimulator, error) {
	if chainID == 0 {
		chainID = valueobject.ChainIDBase
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:     entityPool.Address,
				Exchange:    entityPool.Exchange,
				Type:        entityPool.Type,
				Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, _ int) string { return item.Address }),
				Reserves:    lo.Map(entityPool.Reserves, func(item string, _ int) *big.Int { return bignumber.NewBig(item) }),
				BlockNumber: entityPool.BlockNumber,
			},
		},
		chainID:        chainID,
		periphery:      staticExtra.PeripheryAddress,
		permit2:        lo.Ternary(staticExtra.Permit2Address != "", staticExtra.Permit2Address, defaultPermit2Address),
		wrappedNative:  staticExtra.WrappedNative,
		rawTokenX:      staticExtra.RawTokenX,
		rawTokenY:      staticExtra.RawTokenY,
		priceX96:       extra.PX96,
		feeQ48:         extra.Fee,
		latestBlock:    extra.LatestUpdateBlock,
		concentrationK: extra.ConcentrationK,
		paused:         extra.Paused,
		reserves: lo.Map(entityPool.Reserves, func(item string, _ int) *uint256.Int {
			return big256.New(item)
		}),
		gas: defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}
	if s.paused {
		return nil, ErrPoolPaused
	}
	if s.priceX96 == nil || s.priceX96.IsZero() {
		return nil, ErrZeroPrice
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow || amountIn.Sign() <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	poolParams := &PoolParams{
		SqrtPriceX96:   s.priceX96,
		FeeQ48:         s.feeQ48,
		ReserveX:       s.reserves[0],
		ReserveY:       s.reserves[1],
		ConcentrationK: s.concentrationK,
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
		Gas:            s.gas,
		SwapInfo:       SwapInfo{NextPX96: result.SqrtPriceNext},
	}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	if s.priceX96 != nil {
		cloned.priceX96 = new(uint256.Int).Set(s.priceX96)
	}
	cloned.reserves = lo.Map(s.reserves, func(item *uint256.Int, _ int) *uint256.Int {
		if item == nil {
			return nil
		}

		return new(uint256.Int).Set(item)
	})
	cloned.Info.Reserves = lo.Map(cloned.reserves, func(item *uint256.Int, _ int) *big.Int {
		return item.ToBig()
	})

	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	inAmount := uint256.MustFromBig(params.TokenAmountIn.Amount)
	outAmount := uint256.MustFromBig(params.TokenAmountOut.Amount)

	s.reserves[indexIn] = new(uint256.Int).Add(s.reserves[indexIn], inAmount)
	s.reserves[indexOut] = new(uint256.Int).Sub(s.reserves[indexOut], outAmount)
	s.Info.Reserves[indexIn] = s.reserves[indexIn].ToBig()
	s.Info.Reserves[indexOut] = s.reserves[indexOut].ToBig()

	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok && swapInfo.NextPX96 != nil {
		s.priceX96 = new(uint256.Int).Set(swapInfo.NextPX96)
	}
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return PoolMeta{
		BlockNumber:     s.Info.BlockNumber,
		RouterAddress:   s.periphery,
		Permit2Address:  s.permit2,
		ApprovalAddress: s.GetApprovalAddress(tokenIn, tokenOut),
	}
}

func (s *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	if valueobject.IsNative(tokenIn) || valueobject.IsWrappedNative(tokenIn, s.chainID) {
		return ""
	}

	return s.permit2
}

func (s *PoolSimulator) CanSwapTo(address string) []string {
	if s.GetTokenIndex(address) < 0 {
		return nil
	}

	return lo.Filter(s.Info.Tokens, func(token string, _ int) bool {
		return !strings.EqualFold(token, address)
	})
}

func (s *PoolSimulator) CanSwapFrom(address string) []string {
	return s.CanSwapTo(address)
}
