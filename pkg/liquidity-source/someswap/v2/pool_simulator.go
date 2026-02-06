package someswapv2

import (
	"fmt"
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
	reserves       []*uint256.Int
	baseFee        *uint256.Int
	dynBps         *uint256.Int
	wToken0        *uint256.Int
	wToken1        *uint256.Int
	router         string
	token0, token1 string
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	staticExtra := StaticExtra{
		BaseFee: 0,
		WToken0: 0,
		WToken1: uint32(bpsDen.Uint64()),
	}
	if ep.StaticExtra != "" {
		if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
			return nil, err
		}
	}

	var extra Extra
	if ep.Extra != "" {
		_ = json.Unmarshal([]byte(ep.Extra), &extra)
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(pt *entity.PoolToken, _ int) string { return pt.Address }),
			Reserves:    lo.Map(ep.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
			BlockNumber: ep.BlockNumber,
		}},
		reserves: lo.Map(ep.Reserves, func(r string, _ int) *uint256.Int { return uint256.MustFromDecimal(r) }),
		baseFee:  uint256.NewInt(uint64(staticExtra.BaseFee)),
		dynBps:   uint256.NewInt(uint64(extra.DynBps)),
		wToken0:  uint256.NewInt(uint64(staticExtra.WToken0)),
		wToken1:  uint256.NewInt(uint64(staticExtra.WToken1)),
		router:   staticExtra.Router,
		token0:   staticExtra.Token0,
		token1:   staticExtra.Token1,
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
		Gas: defaultGas,
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

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	tokenInIndex := s.GetTokenIndex(tokenIn)
	return PoolMeta{
		BaseFee:  uint32(s.baseFee.Uint64()),
		WToken0:  uint32(s.wToken0.Uint64()),
		WToken1:  uint32(s.wToken1.Uint64()),
		Router:   s.router,
		TokenIn:  lo.Ternary(tokenInIndex == 0, s.token0, s.token1),
		TokenOut: lo.Ternary(tokenInIndex == 0, s.token1, s.token0),
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

	weight := s.wToken0
	if tokenInIndex == 1 {
		weight = s.wToken1
	}

	totalFee := new(uint256.Int).Add(s.baseFee, s.dynBps)
	maxFee := new(uint256.Int).Sub(bpsDen, uint256.NewInt(1))
	if totalFee.Cmp(maxFee) > 0 {
		totalFee = maxFee
	}

	inFee := new(uint256.Int).Mul(totalFee, weight)
	inFee.Div(inFee, bpsDen)

	outFee := new(uint256.Int)
	if totalFee.Cmp(inFee) > 0 {
		outFee = new(uint256.Int).Sub(totalFee, inFee)
	}

	feeMultiplier := new(uint256.Int).Sub(bpsDen, inFee)
	effIn := new(uint256.Int).Mul(amountIn, feeMultiplier)
	effIn.Div(effIn, bpsDen)
	if effIn.Sign() <= 0 {
		return nil, nil, nil, fmt.Errorf("amount in too small")
	}

	inFeeAmount := new(uint256.Int).Mul(amountIn, inFee)
	inFeeAmount.Div(inFeeAmount, bpsDen)

	outTotal := new(uint256.Int).Mul(effIn, reserveOut)
	denom := new(uint256.Int).Add(reserveIn, effIn)
	outTotal.Div(outTotal, denom)
	if outTotal.Sign() <= 0 {
		return nil, nil, nil, fmt.Errorf("invalid total out")
	}

	outFeeMultiplier := new(uint256.Int).Sub(bpsDen, outFee)
	netOut := new(uint256.Int).Mul(outTotal, outFeeMultiplier)
	netOut.Div(netOut, bpsDen)
	if netOut.Sign() <= 0 {
		return nil, nil, nil, fmt.Errorf("invalid net out")
	}

	return netOut, outTotal, inFeeAmount, nil
}
