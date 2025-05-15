package skypsm

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sky "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maker/savingsdai"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	rate           *uint256.Int
	usdcPrecision  *uint256.Int
	usdsPrecision  *uint256.Int
	susdsPrecision *uint256.Int
	balances       []*uint256.Int

	gas Gas
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(p entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
			Address:     p.Address,
			Exchange:    p.Exchange,
			Type:        p.Type,
			Tokens:      lo.Map(p.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(p.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: p.BlockNumber,
		}},
		rate:           extra.Rate,
		usdcPrecision:  big256.TenPowInt(p.Tokens[0].Decimals),
		usdsPrecision:  big256.TenPowInt(p.Tokens[1].Decimals),
		susdsPrecision: big256.TenPowInt(p.Tokens[2].Decimals),
		balances: lo.Map(p.Reserves, func(reserve string, _ int) *uint256.Int {
			bal, _ := uint256.FromDecimal(reserve)
			return bal
		}),
		gas: defaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn, indexOut := p.GetTokenIndex(params.TokenAmountIn.Token), p.GetTokenIndex(params.TokenOut)
	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)

	amountOut, err := p.getSwapQuote(indexIn, indexOut, amountIn, false)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &poolpkg.TokenAmount{Token: params.TokenAmountIn.Token, Amount: integer.Zero()},
		Gas:            p.gas.SwapExactIn,
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	indexOut, indexIn := p.GetTokenIndex(params.TokenAmountOut.Token), p.GetTokenIndex(params.TokenIn)
	amountOut := uint256.MustFromBig(params.TokenAmountOut.Amount)

	amountIn, err := p.getSwapQuote(indexOut, indexIn, amountOut, true)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: p.Info.Tokens[indexIn], Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: p.Info.Tokens[indexOut], Amount: integer.Zero()},
		Gas:           p.gas.SwapExactIn,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(_ poolpkg.UpdateBalanceParams) {}

func (p *PoolSimulator) GetMetaInfo(_, _ string) interface{} {
	return PoolMeta{
		BlockNumber: p.Info.BlockNumber,
	}
}

func (p *PoolSimulator) getSwapQuote(inIdx, outIdx int, amount *uint256.Int, calcAmtIn bool) (*uint256.Int, error) {
	if calcAmtIn && p.balances[inIdx].Cmp(amount) < 0 {
		return nil, ErrInsufficientBalance
	}

	var (
		quoteAmount *uint256.Int
		err         error
	)
	switch {
	case inIdx == 0 && outIdx == 1:
		quoteAmount, err = p.convertOneToOne(amount, p.usdcPrecision, p.usdsPrecision, calcAmtIn)
	case inIdx == 0 && outIdx == 2:
		quoteAmount, err = p.convertToSUSDS(amount, p.usdcPrecision, calcAmtIn)
	case inIdx == 1 && outIdx == 0:
		quoteAmount, err = p.convertOneToOne(amount, p.usdsPrecision, p.usdcPrecision, calcAmtIn)
	case inIdx == 1 && outIdx == 2:
		quoteAmount, err = p.convertToSUSDS(amount, p.usdsPrecision, calcAmtIn)
	case inIdx == 2 && outIdx == 0:
		quoteAmount, err = p.convertFromSUSDS(amount, p.usdcPrecision, calcAmtIn)
	case inIdx == 2 && outIdx == 1:
		quoteAmount, err = p.convertFromSUSDS(amount, p.usdsPrecision, calcAmtIn)
	default:
		return nil, ErrInvalidToken
	}

	if !calcAmtIn && p.balances[outIdx].Cmp(quoteAmount) < 0 {
		return nil, ErrInsufficientBalance
	}

	return quoteAmount, err
}

func (p *PoolSimulator) convertOneToOne(
	amountIn, assetPrecision, convertAssetPrecision *uint256.Int, roundUp bool) (*uint256.Int, error) {
	var amountOut uint256.Int
	if !roundUp {
		if _, overflow := amountOut.MulDivOverflow(amountIn, convertAssetPrecision, assetPrecision); overflow {
			return nil, number.ErrOverflow
		}
		return &amountOut, nil
	}
	return CeilDiv(amountOut.Mul(amountIn, convertAssetPrecision), assetPrecision)
}

func (p *PoolSimulator) convertToSUSDS(amountIn, assetPrecision *uint256.Int, roundUp bool) (*uint256.Int, error) {
	var amountOut uint256.Int
	if !roundUp {
		if _, overflow := amountOut.MulDivOverflow(amountIn, sky.RAY, p.rate); overflow {
			return nil, number.ErrOverflow
		}
		if _, overflow := amountOut.MulDivOverflow(&amountOut, p.susdsPrecision, assetPrecision); overflow {
			return nil, number.ErrOverflow
		}
		return &amountOut, nil
	}
	temp, err := CeilDiv(amountOut.Mul(amountIn, sky.RAY), p.rate)
	if err != nil {
		return nil, err
	}
	return CeilDiv(temp.Mul(temp, p.susdsPrecision), assetPrecision)
}

func (p *PoolSimulator) convertFromSUSDS(amountIn, assetPrecision *uint256.Int, roundUp bool) (*uint256.Int, error) {
	var amountOut uint256.Int
	if !roundUp {
		if _, overflow := amountOut.MulDivOverflow(amountIn, p.rate, sky.RAY); overflow {
			return nil, number.ErrOverflow
		}
		if _, overflow := amountOut.MulDivOverflow(&amountOut, assetPrecision, p.susdsPrecision); overflow {
			return nil, number.ErrOverflow
		}
		return &amountOut, nil
	}
	temp, err := CeilDiv(amountOut.Mul(amountIn, p.rate), sky.RAY)
	if err != nil {
		return nil, err
	}
	return CeilDiv(temp.Mul(temp, assetPrecision), p.susdsPrecision)
}
