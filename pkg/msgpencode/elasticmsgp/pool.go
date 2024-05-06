//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple Ether Native CurrencyEnum BaseCurrency Token Fraction Price Tick TickData LinkedListData Pool
//msgp:ignore FeeAmount intAsStr
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt
//msgp:shim FeeAmount as:uint64 using:uint64/FeeAmount
//msgp:shim common.Address as:[]byte using:(common.Address).Bytes/common.BytesToAddress
//msgp:shim intAsStr as:string using:intToString/stringToInt

package elasticmsgp

import (
	"math/big"
	"strconv"

	elasticconstants "github.com/KyberNetwork/elastic-go-sdk/v2/constants"
	elasticentities "github.com/KyberNetwork/elastic-go-sdk/v2/entities"
	"github.com/daoleno/uniswap-sdk-core/entities"
	"github.com/ethereum/go-ethereum/common"
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
	LiquidityGross *big.Int
	LiquidityNet   *big.Int
}

type TickData struct {
	LiquidityGross *big.Int
	LiquidityNet   *big.Int
}

type LinkedListData struct {
	Previous int
	Next     int
}

type intAsStr int

func intToString(i intAsStr) string { return strconv.FormatInt(int64(i), 10) }
func stringToInt(s string) intAsStr { i, _ := strconv.ParseInt(s, 10, 64); return intAsStr(i) }

type Pool struct {
	Token0             *Token
	Token1             *Token
	Fee                FeeAmount
	SqrtP              *big.Int
	BaseL              *big.Int
	ReinvestL          *big.Int
	CurrentTick        int
	NearestCurrentTick int
	Ticks              map[intAsStr]TickData
	InitializedTicks   map[intAsStr]LinkedListData

	token0Price *Price
	token1Price *Price
}

func (p *Pool) toSdk() *elasticentities.Pool {
	sdk := &elasticentities.Pool{}
	exported := exportPool(sdk)
	if p.Token0 != nil {
		exported.Token0 = p.Token0.toSdk()
	}
	if p.Token1 != nil {
		exported.Token1 = p.Token1.toSdk()
	}
	exported.Fee = elasticconstants.FeeAmount(p.Fee)
	exported.SqrtP = p.SqrtP
	exported.BaseL = p.BaseL
	exported.ReinvestL = p.ReinvestL
	exported.CurrentTick = p.CurrentTick
	exported.NearestCurrentTick = p.NearestCurrentTick
	exported.Ticks = make(map[int]elasticentities.TickData, len(p.Ticks))
	for index, tick := range p.Ticks {
		exported.Ticks[int(index)] = elasticentities.TickData{
			LiquidityGross: tick.LiquidityGross,
			LiquidityNet:   tick.LiquidityNet,
		}
	}
	exported.InitializedTicks = make(map[int]elasticentities.LinkedListData, len(p.InitializedTicks))
	for index, tick := range p.InitializedTicks {
		exported.InitializedTicks[int(index)] = elasticentities.LinkedListData{
			Previous: tick.Previous,
			Next:     tick.Next,
		}
	}
	if p.token0Price != nil {
		exported.token0Price = p.token0Price.toSdk()
	}
	if p.token1Price != nil {
		exported.token1Price = p.token1Price.toSdk()
	}
	return sdk
}

func (p *Pool) fromSdk(sdk *elasticentities.Pool) *Pool {
	exported := exportPool(sdk)
	if exported.Token0 != nil {
		p.Token0 = new(Token).fromSdk(exported.Token0)
	}
	if exported.Token1 != nil {
		p.Token1 = new(Token).fromSdk(exported.Token1)
	}
	p.Fee = FeeAmount(exported.Fee)
	p.SqrtP = exported.SqrtP
	p.BaseL = exported.BaseL
	p.ReinvestL = exported.ReinvestL
	p.CurrentTick = exported.CurrentTick
	p.NearestCurrentTick = exported.NearestCurrentTick
	p.Ticks = make(map[intAsStr]TickData, len(exported.Ticks))
	for index, tick := range exported.Ticks {
		p.Ticks[intAsStr(index)] = TickData{
			LiquidityGross: tick.LiquidityGross,
			LiquidityNet:   tick.LiquidityNet,
		}
	}
	p.InitializedTicks = make(map[intAsStr]LinkedListData, len(exported.InitializedTicks))
	for index, tick := range exported.InitializedTicks {
		p.InitializedTicks[intAsStr(index)] = LinkedListData{
			Previous: tick.Previous,
			Next:     tick.Next,
		}
	}
	if exported.token0Price != nil {
		p.token0Price = new(Price).fromSdk(exported.token0Price)
	}
	if exported.token1Price != nil {
		p.token1Price = new(Price).fromSdk(exported.token1Price)
	}
	return p
}
