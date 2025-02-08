package platypus

import (
	"math/big"
	"strings"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type (
	PoolSimulator struct {
		pool.Pool

		C1             *big.Int
		RetentionRatio *big.Int
		SlippageParamK *big.Int
		SlippageParamN *big.Int
		XThreshold     *big.Int
		HaircutRate    *big.Int
		SAvaxRate      *big.Int
		AssetByToken   map[string]Asset
		ChainID        valueobject.ChainID
		gas            Gas
	}
)

var _ = pool.RegisterFactory1(DexTypePlatypus, NewPoolSimulator)
var _ = pool.RegisterFactory1(PoolTypePlatypusBase, NewPoolSimulator)
var _ = pool.RegisterFactory1(PoolTypePlatypusAvax, NewPoolSimulator)
var _ = pool.RegisterFactory1(PoolTypePlatypusPure, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	if extra.Paused {
		return nil, ErrPoolPaused
	}

	tokens := make([]string, 0, len(entityPool.Tokens))
	for _, modelPoolToken := range entityPool.Tokens {
		tokens = append(tokens, modelPoolToken.Address)
	}

	info := pool.PoolInfo{
		Address:  entityPool.Address,
		Exchange: entityPool.Exchange,
		Type:     entityPool.Type,
		Tokens:   tokens,
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: info,
		},
		C1:             extra.C1,
		RetentionRatio: extra.RetentionRatio,
		SlippageParamK: extra.SlippageParamK,
		SlippageParamN: extra.SlippageParamN,
		XThreshold:     extra.XThreshold,
		HaircutRate:    extra.HaircutRate,
		AssetByToken:   extra.AssetByToken,
		SAvaxRate:      extra.SAvaxRate,
		ChainID:        chainID,
		gas:            DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	if tokenAmountIn.Token == tokenOut {
		return &pool.CalcAmountOutResult{}, ErrSameAddress
	}

	if tokenAmountIn.Amount.Cmp(bignumber.ZeroBI) <= 0 {
		return &pool.CalcAmountOutResult{}, ErrZeroFromAmount
	}

	fromAsset, err := p._assetOf(tokenAmountIn.Token)
	if err != nil {
		return &pool.CalcAmountOutResult{}, err
	}

	toAsset, err := p._assetOf(tokenOut)
	if err != nil {
		return &pool.CalcAmountOutResult{}, err
	}

	if fromAsset.AggregateAccount != toAsset.AggregateAccount {
		return &pool.CalcAmountOutResult{}, ErrDiffAggAcc
	}

	actualToAmount, hairCut, err := p._quoteFrom(fromAsset, toAsset, tokenAmountIn.Amount)
	if err != nil {
		return &pool.CalcAmountOutResult{}, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: actualToAmount,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: hairCut,
		}, Gas: p.gas.Swap}, nil
}

func (p *PoolSimulator) UpdateBalance(
	params pool.UpdateBalanceParams,
) {
	fromAsset, err := p._assetOf(params.TokenAmountIn.Token)
	if err != nil {
		return
	}

	toAsset, err := p._assetOf(params.TokenAmountOut.Token)
	if err != nil {
		return
	}

	fromAsset.addCash(params.TokenAmountIn.Amount)
	toAsset.removeCash(params.TokenAmountOut.Amount)
	toAsset.addLiability(_dividend(params.Fee.Amount, p.RetentionRatio))

	p.AssetByToken[params.TokenAmountIn.Token] = fromAsset
	p.AssetByToken[params.TokenAmountOut.Token] = toAsset
}

func (p *PoolSimulator) GetMidPrice(
	tokenIn string,
	tokenOut string,
	base *big.Int,
) *big.Int {
	fromAsset, err := p._assetOf(tokenIn)
	if err != nil {
		return bignumber.ZeroBI
	}

	toAsset, err := p._assetOf(tokenOut)
	if err != nil {
		return bignumber.ZeroBI
	}

	actualToAmount, _, err := p._quoteFrom(fromAsset, toAsset, base)
	if err != nil {
		return bignumber.ZeroBI
	}

	return actualToAmount
}

func (p *PoolSimulator) CalcExactQuote(
	tokenIn string,
	tokenOut string,
	base *big.Int,
) *big.Int {
	fromAsset, err := p._assetOf(tokenIn)
	if err != nil {
		return bignumber.ZeroBI
	}

	toAsset, err := p._assetOf(tokenOut)
	if err != nil {
		return bignumber.ZeroBI
	}

	actualToAmount, _, err := p._quoteFrom(fromAsset, toAsset, base)
	if err != nil {
		return bignumber.ZeroBI
	}

	return actualToAmount
}

func (p *PoolSimulator) GetMetaInfo(
	tokenIn string,
	tokenOut string,
) interface{} {
	return nil
}

// _quoteFrom quotes the actual amount user would receive in a swap, taking in account slippage and haircut
// https://github.com/platypus-finance/core/blob/ce7a98d5a12aa54d3f9b31777b6dde8f1f771318/contracts/pool/Pool.sol#L790
func (p *PoolSimulator) _quoteFrom(
	fromAsset Asset,
	toAsset Asset,
	fromAmount *big.Int,
) (*big.Int, *big.Int, error) {
	idealToAmount, err := p._quoteIdealToAmount(fromAsset, toAsset, fromAmount)
	if err != nil {
		return nil, nil, err
	}

	if toAsset.Cash.Cmp(idealToAmount) < 0 {
		return nil, nil, ErrInsufficientCash
	}

	slippageFrom, err := _slippage(
		p.SlippageParamK,
		p.SlippageParamN,
		p.C1,
		p.XThreshold,
		fromAsset.Cash,
		fromAsset.Liability,
		fromAmount,
		true,
	)
	if err != nil {
		return nil, nil, err
	}

	slippageTo, err := _slippage(
		p.SlippageParamK,
		p.SlippageParamN,
		p.C1,
		p.XThreshold,
		toAsset.Cash,
		toAsset.Liability,
		idealToAmount,
		false,
	)
	if err != nil {
		return nil, nil, err
	}

	swappingSlippage := _swappingSlippage(slippageFrom, slippageTo)
	toAmount := wmul(idealToAmount, swappingSlippage)
	haircut := _haircut(toAmount, p.HaircutRate)
	actualToAmount := new(big.Int).Sub(toAmount, haircut)

	return actualToAmount, haircut, nil
}

// _quoteIdealToAmount quotes the ideal amount in case of swap
// https://github.com/platypus-finance/core/blob/ce7a98d5a12aa54d3f9b31777b6dde8f1f771318/contracts/pool/Pool.sol#L832
func (p *PoolSimulator) _quoteIdealToAmount(
	fromAsset Asset,
	toAsset Asset,
	fromAmount *big.Int,
) (*big.Int, error) {
	if p.Info.Type == PoolTypePlatypusAvax {
		return p._quoteIdealToAmountSAvax(fromAsset, toAsset, fromAmount)
	}

	return new(big.Int).Div(
		new(big.Int).Mul(
			fromAmount,
			bignumber.TenPowInt(toAsset.Decimals),
		),
		bignumber.TenPowInt(fromAsset.Decimals),
	), nil
}

// _quoteIdealToAmountSAvax quotes the ideal amount in case of swap for sAvax pool
// https://github.com/platypus-finance/core/blob/ce7a98d5a12aa54d3f9b31777b6dde8f1f771318/contracts/pool/PoolSAvax.sol#L939
func (p *PoolSimulator) _quoteIdealToAmountSAvax(
	fromAsset Asset,
	toAsset Asset,
	fromAmount *big.Int,
) (*big.Int, error) {
	native, ok := valueobject.WrappedNativeMap[p.ChainID]
	if !ok {
		return nil, ErrWrappedNativeNotFound
	}

	fromToken := fromAsset.UnderlyingToken
	toToken := toAsset.UnderlyingToken

	if strings.EqualFold(toToken, native) {
		return wmul(fromAmount, p.SAvaxRate), nil
	}

	if strings.EqualFold(fromToken, native) {
		return wdiv(fromAmount, p.SAvaxRate)
	}

	return nil, ErrUnsupportedSwap
}

// _assetOf gets Asset corresponding to ERC20 token. Returns error if asset does not exist in Pool.
// https://github.com/platypus-finance/core/blob/ce7a98d5a12aa54d3f9b31777b6dde8f1f771318/contracts/pool/Pool.sol#L469
func (p *PoolSimulator) _assetOf(
	token string,
) (Asset, error) {
	asset, ok := p.AssetByToken[token]
	if !ok {
		return Asset{}, ErrAssetNotExist
	}

	return asset, nil
}
