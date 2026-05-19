package brownfiv3

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	approvalInfo         pool.ApprovalInfo
	staticExtra          StaticExtra
	extra                Extra
	reserves             [2]*uint256.Int
	token0Dec, token1Dec uint8
	token0, token1       string
}

var _ = pool.RegisterFactory(DexType, NewPoolSimulator)

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	entityPool := params.EntityPool
	if time.Since(time.Unix(entityPool.Timestamp, 0)) > maxAge {
		return nil, ErrInvalidPrices
	}
	router, ok := Router[params.ChainID]
	if !ok {
		return nil, pool.ErrUnsupported
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	if len(entityPool.Tokens) != 2 {
		return nil, ErrInvalidToken
	}
	if len(entityPool.Reserves) != 2 {
		return nil, ErrInvalidReserve
	}
	if extra.KB == nil || extra.KQ == nil {
		return nil, ErrInvalidPrices
	}
	if extra.Price0 == nil || extra.Price1 == nil {
		return nil, ErrInvalidPrices
	}

	var reserves [2]*uint256.Int
	bigReserves := make([]*big.Int, 2)
	for i := range 2 {
		r, err := uint256.FromDecimal(entityPool.Reserves[i])
		if err != nil {
			return nil, ErrInvalidReserve
		}
		reserves[i] = r
		bigReserves[i] = r.ToBig()
	}

	tokens := []string{entityPool.Tokens[0].Address, entityPool.Tokens[1].Address}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    bigReserves,
			BlockNumber: entityPool.BlockNumber,
		}},
		approvalInfo: pool.ApprovalInfo{ApprovalAddress: hexutil.Encode(router[:])},
		staticExtra:  staticExtra,
		extra:        extra,
		reserves:     reserves,
		token0Dec:    entityPool.Tokens[0].Decimals,
		token1Dec:    entityPool.Tokens[1].Decimals,
		token0:       entityPool.Tokens[0].Address,
		token1:       entityPool.Tokens[1].Address,
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

	amountOut, err := s.doCalc(indexIn, indexOut, amountIn, true)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[indexIn], Amount: bignumber.ZeroBI},
		Gas:            defaultGas,
		SwapInfo:       SwapInfo{PriceUpdateData: s.extra.PriceUpdateData},
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
	}

	amountIn, err := s.doCalc(indexIn, indexOut, amountOut, false)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: s.Info.Tokens[indexIn], Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: s.Info.Tokens[indexOut], Amount: bignumber.ZeroBI},
		Gas:           defaultGas,
	}, nil
}

func (s *PoolSimulator) doCalc(indexIn, indexOut int, input *uint256.Int, isCalcOut bool) (*uint256.Int, error) {
	priceIn, priceOut, adjPrice, kappa, isSell, err := s.swapContext(indexOut)
	if err != nil {
		return nil, err
	}
	inDec, outDec := s.tokenDecimals(indexIn), s.tokenDecimals(indexOut)

	return lo.Ternary(isCalcOut, calcAmountOut, calcAmountIn)(
		inDec, outDec,
		input, s.reserves[indexIn], s.reserves[indexOut],
		priceIn, priceOut, adjPrice, kappa,
		s.extra.Gamma, s.extra.Fee, isSell,
	)
}

// swapContext calls computeSwapPrices off-chain with the current reserves to derive
// direction-aware priceIn/priceOut (Q64 dollar prices) and adjPrice (Q64 relative price).
func (s *PoolSimulator) swapContext(indexOut int) (priceIn, priceOut, adjPrice, kappa *uint256.Int, isSell bool,
	err error) {
	// isSell = tokenOut is the base token
	// quoteTokenIndex==0 → token0=quote, token1=base → SELL iff indexOut==1
	// quoteTokenIndex==1 → token1=quote, token0=base → SELL iff indexOut==0
	if s.staticExtra.QuoteTokenIndex == 0 {
		isSell = indexOut == 1
	} else {
		isSell = indexOut == 0
	}

	quoteIdx := int(s.staticExtra.QuoteTokenIndex)
	baseIdx := 1 - quoteIdx

	reserveBase18 := parseRawToDefaultDecimals(s.reserves[baseIdx], s.tokenDecimals(baseIdx))
	reserveQuote18 := parseRawToDefaultDecimals(s.reserves[quoteIdx], s.tokenDecimals(quoteIdx))

	conf0, conf1 := s.extra.Conf0, s.extra.Conf1
	if conf0 == nil {
		conf0 = new(uint256.Int)
	}
	if conf1 == nil {
		conf1 = new(uint256.Int)
	}

	priceIn, priceOut, adjPrice, err = computeSwapPrices(
		s.extra.Price0, s.extra.Price1, conf0, conf1,
		s.extra.AmmPrice,
		reserveBase18, reserveQuote18,
		s.staticExtra.QuoteTokenIndex,
		s.extra.PythWeight, s.extra.FixS, s.extra.Compress,
		s.extra.SSell, s.extra.SBuy, s.extra.SBound, s.extra.DisThreshold,
		s.extra.Lambda,
		s.extra.Fee,
		isSell,
	)
	if err != nil {
		return nil, nil, nil, nil, false, err
	}

	if isSell {
		kappa = s.extra.KB
	} else {
		kappa = s.extra.KQ
	}
	return
}

func (s *PoolSimulator) tokenDecimals(index int) uint8 {
	if index == 0 {
		return s.token0Dec
	}
	return s.token1Dec
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
	amtIn := uint256.MustFromBig(params.TokenAmountIn.Amount)
	amtOut := uint256.MustFromBig(params.TokenAmountOut.Amount)
	s.reserves[indexIn] = amtIn.Add(s.reserves[indexIn], amtIn)
	s.reserves[indexOut] = amtOut.Sub(s.reserves[indexOut], amtOut)
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return PoolMeta{
		ApprovalInfo: s.approvalInfo,
		Fee:          s.extra.Fee,
	}
}
