package someswapv2

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
	reserves []*uint256.Int
	baseFee  *uint256.Int
	wToken0  *uint256.Int
	wToken1  *uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	staticExtra := StaticExtra{
		BaseFee: 0,
		WToken0: 0,
		WToken1: uint32(weightDen.Uint64()),
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

	baseFee := uint256.NewInt(uint64(staticExtra.BaseFee))
	wToken0 := uint256.NewInt(uint64(staticExtra.WToken0))
	wToken1 := uint256.NewInt(uint64(staticExtra.WToken1))

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
		Pool:     pool.Pool{Info: info},
		reserves: reserves,
		baseFee:  baseFee,
		wToken0:  wToken0,
		wToken1:  wToken1,
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
		Gas: 80000,
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
		BaseFee: uint32(s.baseFee.Uint64()),
		WToken0: uint32(s.wToken0.Uint64()),
		WToken1: uint32(s.wToken1.Uint64()),
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

	feeDenU256 := uint256.MustFromBig(feeDen)
	weightDenU256 := uint256.MustFromBig(weightDen)

	weight := s.wToken0
	if tokenInIndex == 1 {
		weight = s.wToken1
	}

	inFee := new(uint256.Int).Mul(s.baseFee, weight)
	inFee.Div(inFee, weightDenU256)

	outFee := new(uint256.Int)
	if s.baseFee.Cmp(inFee) > 0 {
		outFee = new(uint256.Int).Sub(s.baseFee, inFee)
	}

	feeMultiplier := new(uint256.Int).Sub(feeDenU256, inFee)
	effIn := new(uint256.Int).Mul(amountIn, feeMultiplier)
	effIn.Div(effIn, feeDenU256)
	if effIn.Sign() <= 0 {
		return nil, nil, nil, fmt.Errorf("amount in too small")
	}

	inFeeAmount := new(uint256.Int).Mul(amountIn, inFee)
	inFeeAmount.Div(inFeeAmount, feeDenU256)

	outTotal := new(uint256.Int).Mul(effIn, reserveOut)
	denom := new(uint256.Int).Add(reserveIn, effIn)
	outTotal.Div(outTotal, denom)
	if outTotal.Sign() <= 0 {
		return nil, nil, nil, fmt.Errorf("invalid total out")
	}

	outFeeMultiplier := new(uint256.Int).Sub(feeDenU256, outFee)
	netOut := new(uint256.Int).Mul(outTotal, outFeeMultiplier)
	netOut.Div(netOut, feeDenU256)
	if netOut.Sign() <= 0 {
		return nil, nil, nil, fmt.Errorf("invalid net out")
	}

	return netOut, outTotal, inFeeAmount, nil
}
