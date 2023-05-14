package platypus

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type (
	Pool struct {
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

	Extra struct {
		C1             *big.Int         `json:"c1"`
		HaircutRate    *big.Int         `json:"haircutRate"`
		RetentionRatio *big.Int         `json:"retentionRatio"`
		SlippageParamK *big.Int         `json:"slippageParamK"`
		SlippageParamN *big.Int         `json:"slippageParamN"`
		XThreshold     *big.Int         `json:"xThreshold"`
		SAvaxRate      *big.Int         `json:"sAvaxRate"`
		Paused         bool             `json:"paused"`
		AssetByToken   map[string]Asset `json:"assetByToken"`
	}
)

func NewPool(entityPool entity.Pool, chainID valueobject.ChainID) (*Pool, error) {
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

	return &Pool{
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

func (p *Pool) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	if tokenAmountIn.Token == tokenOut {
		return &pool.CalcAmountOutResult{}, ErrSameAddress
	}

	if tokenAmountIn.Amount.Cmp(constant.Zero) <= 0 {
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

func (p *Pool) UpdateBalance(
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

func (p *Pool) GetLpToken() string {
	return ""
}

func (p *Pool) CanSwapTo(address string) []string {
	isTokenExists := false
	for _, token := range p.Info.Tokens {
		if strings.EqualFold(token, address) {
			isTokenExists = true
		}
	}

	if !isTokenExists {
		return nil
	}

	swappableTokens := make([]string, 0, len(p.Info.Tokens)-1)
	for _, token := range p.Info.Tokens {
		if token == address {
			continue
		}

		swappableTokens = append(swappableTokens, token)
	}

	return swappableTokens
}

func (p *Pool) GetMidPrice(
	tokenIn string,
	tokenOut string,
	base *big.Int,
) *big.Int {
	fromAsset, err := p._assetOf(tokenIn)
	if err != nil {
		return constant.Zero
	}

	toAsset, err := p._assetOf(tokenOut)
	if err != nil {
		return constant.Zero
	}

	actualToAmount, _, err := p._quoteFrom(fromAsset, toAsset, base)
	if err != nil {
		return constant.Zero
	}

	return actualToAmount
}

func (p *Pool) CalcExactQuote(
	tokenIn string,
	tokenOut string,
	base *big.Int,
) *big.Int {
	fromAsset, err := p._assetOf(tokenIn)
	if err != nil {
		return constant.Zero
	}

	toAsset, err := p._assetOf(tokenOut)
	if err != nil {
		return constant.Zero
	}

	actualToAmount, _, err := p._quoteFrom(fromAsset, toAsset, base)
	if err != nil {
		return constant.Zero
	}

	return actualToAmount
}

func (p *Pool) GetMetaInfo(
	tokenIn string,
	tokenOut string,
) interface{} {
	return nil
}

// _quoteFrom quotes the actual amount user would receive in a swap, taking in account slippage and haircut
// https://github.com/platypus-finance/core/blob/ce7a98d5a12aa54d3f9b31777b6dde8f1f771318/contracts/pool/Pool.sol#L790
func (p *Pool) _quoteFrom(
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
func (p *Pool) _quoteIdealToAmount(
	fromAsset Asset,
	toAsset Asset,
	fromAmount *big.Int,
) (*big.Int, error) {
	if p.Info.Type == constant.PoolTypes.PlatypusAvax {
		return p._quoteIdealToAmountSAvax(fromAsset, toAsset, fromAmount)
	}

	return new(big.Int).Div(
		new(big.Int).Mul(
			fromAmount,
			constant.TenPowInt(toAsset.Decimals),
		),
		constant.TenPowInt(fromAsset.Decimals),
	), nil
}

// _quoteIdealToAmountSAvax quotes the ideal amount in case of swap for sAvax pool
// https://github.com/platypus-finance/core/blob/ce7a98d5a12aa54d3f9b31777b6dde8f1f771318/contracts/pool/PoolSAvax.sol#L939
func (p *Pool) _quoteIdealToAmountSAvax(
	fromAsset Asset,
	toAsset Asset,
	fromAmount *big.Int,
) (*big.Int, error) {
	weth, ok := valueobject.WETHByChainID[p.ChainID]
	if !ok {
		return nil, ErrWETHNotFound
	}

	fromToken := fromAsset.UnderlyingToken
	toToken := toAsset.UnderlyingToken

	if strings.EqualFold(toToken, weth) {
		return wmul(fromAmount, p.SAvaxRate), nil
	}

	if strings.EqualFold(fromToken, weth) {
		return wdiv(fromAmount, p.SAvaxRate)
	}

	return nil, ErrUnsupportedSwap
}

// _assetOf gets Asset corresponding to ERC20 token. Returns error if asset does not exist in Pool.
// https://github.com/platypus-finance/core/blob/ce7a98d5a12aa54d3f9b31777b6dde8f1f771318/contracts/pool/Pool.sol#L469
func (p *Pool) _assetOf(
	token string,
) (Asset, error) {
	asset, ok := p.AssetByToken[token]
	if !ok {
		return Asset{}, ErrAssetNotExist
	}

	return asset, nil
}
