package someswap

import (
	"fmt"
	"math/big"
	"slices"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	reserves     []*uint256.Int
	baseFeeBps   *uint256.Int
	dynamicFeeBps *uint256.Int
	wToken0In    *uint256.Int
	wToken1In    *uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	staticExtra := StaticExtra{
		BaseFeeBps:    "0",
		DynamicFeeBps: "0",
		WToken0In:     weightDen.String(),
		WToken1In:     weightDen.String(),
	}
	if entityPool.StaticExtra != "" {
		if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
			return nil, err
		}
	}

	reserves := make([]*uint256.Int, len(entityPool.Reserves))
	for i, reserveStr := range entityPool.Reserves {
		reserve, err := uint256.FromDecimal(reserveStr)
		if err != nil {
			return nil, err
		}
		reserves[i] = reserve
	}

	baseFeeBps, err := uint256.FromDecimal(staticExtra.BaseFeeBps)
	if err != nil {
		return nil, err
	}
	dynamicFeeBps, err := uint256.FromDecimal(staticExtra.DynamicFeeBps)
	if err != nil {
		return nil, err
	}
	wToken0In, err := uint256.FromDecimal(staticExtra.WToken0In)
	if err != nil {
		return nil, err
	}
	wToken1In, err := uint256.FromDecimal(staticExtra.WToken1In)
	if err != nil {
		return nil, err
	}

	info := pool.PoolInfo{
		Address:  entityPool.Address,
		Exchange: entityPool.Exchange,
		Type:     entityPool.Type,
		Tokens: []string{
			entityPool.Tokens[0].Address,
			entityPool.Tokens[1].Address,
		},
		Reserves: []*big.Int{
			bignumber.NewBig(entityPool.Reserves[0]),
			bignumber.NewBig(entityPool.Reserves[1]),
		},
		BlockNumber: entityPool.BlockNumber,
	}

	return &PoolSimulator{
		Pool:         pool.Pool{Info: info},
		reserves:     reserves,
		baseFeeBps:   baseFeeBps,
		dynamicFeeBps: dynamicFeeBps,
		wToken0In:    wToken0In,
		wToken1In:    wToken1In,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	tokenInIndex := s.GetTokenIndex(tokenAmountIn.Token)
	tokenOutIndex := s.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, fmt.Errorf("invalid token index")
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow || amountIn.Sign() <= 0 {
		return nil, fmt.Errorf("invalid amount in")
	}

	netOut, _, _, err := s.calcAmountOutDetailed(amountIn, tokenInIndex, tokenOutIndex)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: netOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: bignumber.ZeroBI,
		},
		Gas: 60000,
	}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.reserves = slices.Clone(s.reserves)
	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenInIndex := s.GetTokenIndex(params.TokenAmountIn.Token)
	tokenOutIndex := s.GetTokenIndex(params.TokenAmountOut.Token)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return
	}

	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)
	netOut, outTotal, inFee, err := s.calcAmountOutDetailed(amountIn, tokenInIndex, tokenOutIndex)
	if err != nil || netOut.Sign() == 0 {
		return
	}

	inDelta := new(uint256.Int).Sub(amountIn, inFee)
	s.reserves[tokenInIndex] = new(uint256.Int).Add(s.reserves[tokenInIndex], inDelta)
	s.reserves[tokenOutIndex] = new(uint256.Int).Sub(s.reserves[tokenOutIndex], outTotal)
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return PoolMeta{
		BaseFeeBps:    s.baseFeeBps.String(),
		DynamicFeeBps: s.dynamicFeeBps.String(),
		WToken0In:     s.wToken0In.String(),
		WToken1In:     s.wToken1In.String(),
	}
}

func (s *PoolSimulator) calcAmountOutDetailed(amountIn *uint256.Int, tokenInIndex, tokenOutIndex int) (*uint256.Int, *uint256.Int, *uint256.Int, error) {
	if amountIn == nil || amountIn.Sign() <= 0 {
		return nil, nil, nil, fmt.Errorf("invalid amount in")
	}

	reserveIn := s.reserves[tokenInIndex]
	reserveOut := s.reserves[tokenOutIndex]
	if reserveIn == nil || reserveOut == nil || reserveIn.Sign() <= 0 || reserveOut.Sign() <= 0 {
		return nil, nil, nil, fmt.Errorf("invalid reserves")
	}

	totalFeeBps := new(uint256.Int).Add(s.baseFeeBps, s.dynamicFeeBps)
	if totalFeeBps.Cmp(uint256.NewInt(9999)) > 0 {
		totalFeeBps = uint256.NewInt(9999)
	}

	weight := s.wToken0In
	if tokenInIndex == 1 {
		weight = s.wToken1In
	}

	inBps := new(uint256.Int).Mul(totalFeeBps, weight)
	inBps.Div(inBps, uint256.MustFromBig(weightDen))
	if inBps.Cmp(totalFeeBps) > 0 {
		inBps = new(uint256.Int).Set(totalFeeBps)
	}
	outBps := new(uint256.Int).Sub(totalFeeBps, inBps)

	inFee := new(uint256.Int).Mul(amountIn, inBps)
	inFee.Div(inFee, uint256.MustFromBig(bpsDen))
	effIn := new(uint256.Int).Sub(amountIn, inFee)
	if effIn.Sign() <= 0 {
		return nil, nil, nil, fmt.Errorf("amount in too small")
	}

	outTotal := new(uint256.Int).Mul(effIn, reserveOut)
	denom := new(uint256.Int).Add(reserveIn, effIn)
	outTotal.Div(outTotal, denom)
	if outTotal.Sign() <= 0 {
		return nil, nil, nil, fmt.Errorf("invalid total out")
	}

	outFee := new(uint256.Int).Mul(outTotal, outBps)
	outFee.Div(outFee, uint256.MustFromBig(bpsDen))
	netOut := new(uint256.Int).Sub(outTotal, outFee)
	if netOut.Sign() <= 0 {
		return nil, nil, nil, fmt.Errorf("invalid net out")
	}

	return netOut, outTotal, inFee, nil
}
