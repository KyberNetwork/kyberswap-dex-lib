package prmfun

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// Token index convention: 0 = desk (wrapped native), 1 = meme. Only ETH-paired,
// Trading-phase MemeCurve pools live here; graduated ones move to the uniswap-v4-premium
// hook, and memes paired against an ERC20 desk are out of scope.
const (
	indexDesk = 0
	indexMeme = 1
)

type PoolSimulator struct {
	pool.Pool

	routerAddress  string
	curveAddress   string
	graduationDesk *uint256.Int

	phase       uint8
	virtualMeme *uint256.Int
	virtualDesk *uint256.Int
	memeSold    *uint256.Int
	deskRaised  *uint256.Int
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

	graduationDesk, err := uint256.FromDecimal(staticExtra.GraduationDesk)
	if err != nil {
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
		routerAddress:  staticExtra.RouterAddress,
		curveAddress:   staticExtra.CurveAddress,
		graduationDesk: graduationDesk,
		phase:          extra.Phase,
		virtualMeme:    extra.VirtualMeme,
		virtualDesk:    extra.VirtualDesk,
		memeSold:       extra.MemeSold,
		deskRaised:     extra.DeskRaised,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	if s.phase != PhaseTrading {
		return nil, ErrPoolNotTrading
	}

	amountIn := uint256.MustFromBig(tokenAmountIn.Amount)
	if amountIn.IsZero() {
		return nil, ErrZeroAmount
	}

	isBuy := indexIn == indexDesk

	if isBuy {
		return s.calcBuy(tokenOut, amountIn)
	}
	return s.calcSell(tokenOut, amountIn)
}

func (s *PoolSimulator) calcBuy(tokenOut string, deskIn *uint256.Int) (*pool.CalcAmountOutResult, error) {
	// Required to avoid an underflow in math.go's room := graduationDesk - deskRaised; a
	// sold-out pool (memeSold == saleSupply) is caught below via result.MemeOut.IsZero().
	if s.deskRaised.Cmp(s.graduationDesk) >= 0 {
		return nil, ErrPoolNotTrading
	}

	result := QuoteBuy(deskIn, s.virtualMeme, s.virtualDesk, s.memeSold, s.deskRaised, s.graduationDesk, uSaleSupply)
	if result.MemeOut.IsZero() || result.DeskUsed.IsZero() {
		return nil, ErrInvalidAmount
	}

	var remainingIn *pool.TokenAmount
	if deskIn.Gt(result.DeskUsed) {
		remainingIn = &pool.TokenAmount{
			Token:  s.Info.Tokens[indexDesk],
			Amount: new(uint256.Int).Sub(deskIn, result.DeskUsed).ToBig(),
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: tokenOut, Amount: result.MemeOut.ToBig()},
		RemainingTokenAmountIn: remainingIn,
		Fee:                    &pool.TokenAmount{Token: s.Info.Tokens[indexDesk], Amount: result.Fee.ToBig()},
		Gas:                    buyGas,
		SwapInfo: &SwapInfo{
			IsBuy:        true,
			CurveAddress: s.curveAddress,
		},
	}, nil
}

func (s *PoolSimulator) calcSell(tokenOut string, memeIn *uint256.Int) (*pool.CalcAmountOutResult, error) {
	if memeIn.Gt(s.memeSold) {
		return nil, ErrInsufficientLiquidity
	}

	gross, result := QuoteSell(memeIn, s.virtualMeme, s.virtualDesk)
	if gross.Gt(s.deskRaised) {
		// Mirrors MemeCurve.sol's quoteSell returning (0, 0) here - the curve does not
		// have enough recorded desk-token principal to honor this sell.
		return nil, ErrInsufficientLiquidity
	}
	if result.DeskOut.IsZero() {
		return nil, ErrInvalidAmount
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: result.DeskOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[indexDesk], Amount: result.Fee.ToBig()},
		Gas:            sellGas,
		SwapInfo: &SwapInfo{
			IsBuy:        false,
			CurveAddress: s.curveAddress,
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(*SwapInfo)
	if !ok {
		return
	}

	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)
	amountOut := uint256.MustFromBig(params.TokenAmountOut.Amount)
	fee := uint256.MustFromBig(params.Fee.Amount)

	if swapInfo.IsBuy {
		net := new(uint256.Int).Sub(amountIn, fee)
		s.deskRaised.Add(s.deskRaised, net)
		s.virtualDesk.Add(s.virtualDesk, net)
		s.virtualMeme.Sub(s.virtualMeme, amountOut)
		s.memeSold.Add(s.memeSold, amountOut)
	} else {
		gross := new(uint256.Int).Add(amountOut, fee)
		s.memeSold.Sub(s.memeSold, amountIn)
		s.virtualMeme.Add(s.virtualMeme, amountIn)
		s.virtualDesk.Sub(s.virtualDesk, gross)
		s.deskRaised.Sub(s.deskRaised, gross)
	}

	if s.deskRaised.Cmp(s.graduationDesk) >= 0 {
		s.phase = PhaseGraduated
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.virtualMeme = new(uint256.Int).Set(s.virtualMeme)
	cloned.virtualDesk = new(uint256.Int).Set(s.virtualDesk)
	cloned.memeSold = new(uint256.Int).Set(s.memeSold)
	cloned.deskRaised = new(uint256.Int).Set(s.deskRaised)
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{
		ApprovalAddress: s.routerAddress,
		BlockNumber:     s.Info.BlockNumber,
	}
}
