//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple Ether Native CurrencyEnum BaseCurrency Token Fraction Price Tick TickListDataProvider TickDataProviderEnum Pool
//msgp:ignore FeeAmount
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt
//msgp:shim *uint256.Int as:[]byte using:msgpencode.EncodeUint256/msgpencode.DecodeUint256
//msgp:shim *utils.Int128 as:[]byte using:msgpencode.EncodeInt256/msgpencode.DecodeInt256
//msgp:shim *utils.Uint160 as:[]byte using:msgpencode.EncodeUint256/msgpencode.DecodeUint256
//msgp:shim *utils.Uint128 as:[]byte using:msgpencode.EncodeUint256/msgpencode.DecodeUint256
//msgp:shim FeeAmount as:uint64 using:uint64/FeeAmount
//msgp:shim common.Address as:[]byte using:(common.Address).Bytes/common.BytesToAddress

package uniswapv3uint256mspg

import (
	"math/big"

	uniswapv3constants "github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	uniswapv3entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/daoleno/uniswap-sdk-core/entities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// Ether is the main usage of a 'native' currency, i.e. for Ethereum mainnet and all testnets
type Ether struct {
	*BaseCurrency
}

func (e *Ether) toSdk() *entities.Ether {
	sdk := &entities.Ether{}
	if e.BaseCurrency != nil {
		sdk.BaseCurrency = e.BaseCurrency.toSdk()
	}
	return sdk
}

func (e *Ether) fromSdk(sdk *entities.Ether) *Ether {
	if sdk.BaseCurrency != nil {
		e.BaseCurrency = new(BaseCurrency).fromSdk(sdk.BaseCurrency)
	}
	return e
}

type Native struct {
	*BaseCurrency
	wrapped *Token
}

func (n *Native) toSdk() *entities.Native {
	sdk := &entities.Native{}
	exported := exportNative(sdk)
	if n.BaseCurrency != nil {
		exported.BaseCurrency = n.BaseCurrency.toSdk()
	}
	if n.wrapped != nil {
		exported.wrapped = n.wrapped.toSdk()
	}
	return sdk
}

func (n *Native) fromSdk(sdk *entities.Native) *Native {
	exported := exportNative(sdk)
	if exported.BaseCurrency != nil {
		n.BaseCurrency = new(BaseCurrency).fromSdk(exported.BaseCurrency)
	}
	if exported.wrapped != nil {
		n.wrapped = new(Token).fromSdk(exported.wrapped)
	}
	return n
}

type CurrencyEnum struct {
	Ether  *Ether
	Native *Native
	Token  *Token
	Base   *BaseCurrency
}

func (c *CurrencyEnum) toSdk() entities.Currency {
	if c.Ether != nil {
		return c.Ether.toSdk()
	}
	if c.Native != nil {
		return c.Native.toSdk()
	}
	if c.Token != nil {
		return c.Token.toSdk()
	}
	if c.Base != nil {
		return c.Base.toSdk()
	}
	return nil
}

func (c *CurrencyEnum) fromSdk(sdk entities.Currency) *CurrencyEnum {
	switch sdk := sdk.(type) {
	case *entities.Ether:
		c.Ether = new(Ether).fromSdk(sdk)
	case *entities.Native:
		c.Native = new(Native).fromSdk(sdk)
	case *entities.Token:
		c.Token = new(Token).fromSdk(sdk)
	case *entities.BaseCurrency:
		c.Base = new(BaseCurrency).fromSdk(sdk)
	}
	return c
}

type BaseCurrency struct {
	isNative bool   // Returns whether the currency is native to the chain and must be wrapped (e.g. Ether)
	isToken  bool   // Returns whether the currency is a token that is usable in Uniswap without wrapping
	chainId  uint   // The chain ID on which this currency resides
	decimals uint   // The decimals used in representing currency amounts
	symbol   string // The symbol of the currency, i.e. a short textual non-unique identifier
	name     string // The name of the currency, i.e. a descriptive textual non-unique identifier
}

func (b *BaseCurrency) fromSdk(s *entities.BaseCurrency) *BaseCurrency {
	b.isNative = s.IsNative()
	b.isToken = s.IsToken()
	b.chainId = s.ChainId()
	b.decimals = s.Decimals()
	b.symbol = s.Symbol()
	b.name = s.Name()
	return b
}

func (b *BaseCurrency) toSdk() *entities.BaseCurrency {
	exported := &baseCurrencyExporter{
		isNative: b.isNative,
		isToken:  b.isToken,
		chainId:  b.chainId,
		decimals: b.decimals,
		symbol:   b.symbol,
		name:     b.name,
	}
	return fromBaseCurrencyExporter(exported)
}

type Token struct {
	*BaseCurrency
	Address common.Address // The contract address on the chain on which this token lives
}

func (t *Token) fromSdk(sdk *entities.Token) *Token {
	if sdk.BaseCurrency != nil {
		t.BaseCurrency = new(BaseCurrency).fromSdk(sdk.BaseCurrency)
	}
	t.Address = sdk.Address
	return t
}

func (t *Token) toSdk() *entities.Token {
	sdk := &entities.Token{}
	if t.BaseCurrency != nil {
		sdk.BaseCurrency = t.BaseCurrency.toSdk()
	}
	sdk.Address = t.Address
	exportBaseCurrency(sdk.BaseCurrency).currency = sdk
	return sdk
}

type FeeAmount uint64

type Fraction struct {
	Numerator   *big.Int
	Denominator *big.Int
}

func (f *Fraction) fromSdk(sdk *entities.Fraction) *Fraction {
	f.Numerator = sdk.Numerator
	f.Denominator = sdk.Denominator
	return f
}

func (f *Fraction) toSdk() *entities.Fraction {
	return &entities.Fraction{
		Numerator:   f.Numerator,
		Denominator: f.Denominator,
	}
}

type Price struct {
	*Fraction
	BaseCurrency  CurrencyEnum // input i.e. denominator
	QuoteCurrency CurrencyEnum // output i.e. numerator
	Scalar        *Fraction    // used to adjust the raw fraction w/r/t the decimals of the {base,quote}Token
}

func (p *Price) fromSdk(sdk *entities.Price) *Price {
	if sdk.Fraction != nil {
		p.Fraction = new(Fraction).fromSdk(sdk.Fraction)
	}
	p.BaseCurrency.fromSdk(sdk.BaseCurrency)
	p.QuoteCurrency.fromSdk(sdk.QuoteCurrency)
	if sdk.Scalar != nil {
		p.Scalar = new(Fraction).fromSdk(sdk.Scalar)
	}
	return p
}

func (p *Price) toSdk() *entities.Price {
	sdk := &entities.Price{}
	if p.Fraction != nil {
		sdk.Fraction = p.Fraction.toSdk()
	}
	sdk.BaseCurrency = p.BaseCurrency.toSdk()
	sdk.QuoteCurrency = p.QuoteCurrency.toSdk()
	if p.Scalar != nil {
		sdk.Scalar = p.Scalar.toSdk()
	}
	return sdk
}

type Tick struct {
	Index          int
	LiquidityGross *uint256.Int
	LiquidityNet   *utils.Int128
}

func (t *Tick) fromSdk(sdk *uniswapv3entities.Tick) *Tick {
	t.Index = sdk.Index
	t.LiquidityGross = sdk.LiquidityGross
	t.LiquidityNet = sdk.LiquidityNet
	return t
}

func (t *Tick) toSdk() *uniswapv3entities.Tick {
	return &uniswapv3entities.Tick{
		Index:          t.Index,
		LiquidityGross: t.LiquidityGross,
		LiquidityNet:   t.LiquidityNet,
	}
}

type TickListDataProvider struct {
	ticks []Tick
}

func (tl *TickListDataProvider) fromSdk(sdk *uniswapv3entities.TickListDataProvider) *TickListDataProvider {
	exported := exportTickListDataProvider(sdk)
	tl.ticks = make([]Tick, len(exported.ticks))
	for i, t := range exported.ticks {
		tl.ticks[i].fromSdk(&t)
	}
	return tl
}

func (tl *TickListDataProvider) toSdk() *uniswapv3entities.TickListDataProvider {
	sdk := new(uniswapv3entities.TickListDataProvider)
	exported := exportTickListDataProvider(sdk)
	exported.ticks = make([]uniswapv3entities.Tick, len(tl.ticks))
	for i, t := range tl.ticks {
		exported.ticks[i] = *t.toSdk()
	}
	return sdk
}

type TickDataProviderEnum struct {
	List *TickListDataProvider `msg:",omitempty"`
}

func (t *TickDataProviderEnum) toSdk() uniswapv3entities.TickDataProvider {
	if t.List != nil {
		return t.List.toSdk()
	}
	return nil
}

func (t *TickDataProviderEnum) fromSdk(sdk uniswapv3entities.TickDataProvider) *TickDataProviderEnum {
	switch sdk := sdk.(type) {
	case *uniswapv3entities.TickListDataProvider:
		t.List = new(TickListDataProvider).fromSdk(sdk)
	}
	return t
}

type Pool struct {
	Token0           *Token
	Token1           *Token
	Fee              FeeAmount
	SqrtRatioX96     *utils.Uint160
	Liquidity        *utils.Uint128
	TickCurrent      int
	TickDataProvider TickDataProviderEnum

	token0Price *Price
	token1Price *Price
}

func (p *Pool) toSdk() *uniswapv3entities.Pool {
	sdk := &uniswapv3entities.Pool{}
	exported := exportPool(sdk)
	if p.Token0 != nil {
		exported.Token0 = p.Token0.toSdk()
	}
	if p.Token1 != nil {
		exported.Token1 = p.Token1.toSdk()
	}
	exported.Fee = uniswapv3constants.FeeAmount(p.Fee)
	exported.SqrtRatioX96 = p.SqrtRatioX96
	exported.Liquidity = p.Liquidity
	exported.TickCurrent = p.TickCurrent
	exported.TickDataProvider = p.TickDataProvider.toSdk()
	if p.token0Price != nil {
		exported.token0Price = p.token0Price.toSdk()
	}
	if p.token1Price != nil {
		exported.token1Price = p.token1Price.toSdk()
	}
	return sdk
}

func (p *Pool) fromSdk(sdk *uniswapv3entities.Pool) *Pool {
	exported := exportPool(sdk)
	if exported.Token0 != nil {
		p.Token0 = new(Token).fromSdk(exported.Token0)
	}
	if exported.Token1 != nil {
		p.Token1 = new(Token).fromSdk(exported.Token1)
	}
	p.Fee = FeeAmount(exported.Fee)
	p.SqrtRatioX96 = exported.SqrtRatioX96
	p.Liquidity = exported.Liquidity
	p.TickCurrent = exported.TickCurrent
	p.TickDataProvider.fromSdk(exported.TickDataProvider)
	if exported.token0Price != nil {
		p.token0Price = new(Price).fromSdk(exported.token0Price)
	}
	if exported.token1Price != nil {
		p.token1Price = new(Price).fromSdk(exported.token1Price)
	}
	return p
}
