package math

import (
	"fmt"

	"github.com/holiman/uint256"
)

// Liquidity is NightlyPoolStorage.Liquidity: a level's balances in native token units.
type Liquidity struct {
	BaseAmount  uint256.Int `json:"baseAmount"`
	QuoteAmount uint256.Int `json:"quoteAmount"`
}

// Level is one price level: its configuration and its balances.
type Level struct {
	Config    PriceLevelConfig `json:"config"`
	Liquidity Liquidity        `json:"liquidity"`
}

// Position is NightlyPoolStorage.PriceLevelPosition: the current level and the fraction inside it.
type Position struct {
	LevelID  int16    `json:"levelId"`
	Fraction Fraction `json:"fraction"`
}

// OraclePrice is NightlyPriceOracle.Price: the mid price with eight decimals, the spreads in
// millionths, and the time of the price.
type OraclePrice struct {
	MidPrice    uint64 `json:"midPrice"`
	BidSpread   Spread `json:"bidSpread"`
	AskSpread   Spread `json:"askSpread"`
	TimestampMs uint64 `json:"timestampMs"`
}

// Quote is NightlyPool.Quote.
type Quote struct {
	// AmountIn is the input that the taker pays, in native units of the input token.
	AmountIn uint256.Int `json:"amountIn"`
	// AmountOut is the output that the taker receives, in native units of the output token.
	AmountOut   uint256.Int `json:"amountOut"`
	EndPosition Position    `json:"endPosition"`
}

// LevelWrite is one level's balances after a swap's step in that level.
type LevelWrite struct {
	LevelID   int16     `json:"levelId"`
	Liquidity Liquidity `json:"liquidity"`
}

// Swap is a walk's result: its Quote, and the level balances that the swap writes. Apply writes
// them and the end position into the pool, as the swap's storage writes.
type Swap struct {
	Quote  Quote        `json:"quote"`
	Writes []LevelWrite `json:"writes"`
}

// Pool is the state of a NightlyPool that a walk reads and Apply writes. Marshal it to JSON
// through a pointer: uint256.Int encodes as a decimal string only through its pointer.
type Pool struct {
	BaseDecimals  uint8  `json:"baseDecimals"`
	QuoteDecimals uint8  `json:"quoteDecimals"`
	MaxPriceAgeMs uint32 `json:"maxPriceAgeMs"`
	LowestLevelID int16  `json:"lowestLevelId"`
	// Levels holds each level from LowestLevelID up to the highest level, without gaps.
	Levels   []Level  `json:"levels"`
	Position Position `json:"position"`
}

// BuyWithExactQuote is NightlyPool._buyWithExactQuote without a rebalance, as swapExactIn and
// quoteSwapExactIn run it: it buys base for exactly quoteIn, walking up from the position at the
// oracle's ask spread. blockTimestamp is the block's time in seconds, against which the price's age
// counts. It writes nothing; Apply writes the returned Swap, as the swap does.
func (p *Pool) BuyWithExactQuote(quoteIn *uint256.Int, oracle OraclePrice, blockTimestamp uint64) (Swap, error) {
	if quoteIn.IsZero() {
		return Swap{}, ErrZeroSwapAmount
	}
	mid, err := p.walkPrice(oracle, blockTimestamp)
	if err != nil {
		return Swap{}, err
	}
	quote := Quote{AmountIn: *quoteIn, EndPosition: p.Position}
	remaining := new(uint256.Int).Set(quoteIn)
	var writes []LevelWrite
	for {
		step, write, err := p.buyFromLevel(quote.EndPosition, remaining, mid, oracle.AskSpread)
		if err != nil {
			return Swap{}, err
		}
		if write != nil {
			writes = append(writes, *write)
		}
		if remaining, err = Sub(remaining, &step.QuoteAmount); err != nil {
			return Swap{}, err
		}
		amountOut, err := Add(&quote.AmountOut, &step.BaseAmount)
		if err != nil {
			return Swap{}, err
		}
		quote.AmountOut = *amountOut
		quote.EndPosition.Fraction = step.End
		// An exact drain ends at the upper edge. Only more offered quote moves to the next level up.
		if remaining.IsZero() {
			break
		}
		if quote.EndPosition.LevelID == p.highestLevelID() {
			return Swap{}, fmt.Errorf("%w: %s left", ErrInsufficientLiquidity, remaining.Dec())
		}
		quote.EndPosition = Position{LevelID: quote.EndPosition.LevelID + 1, Fraction: Lower}
	}
	// Base outputs round down, so a tiny purchase can receive nothing.
	if quote.AmountOut.IsZero() {
		return Swap{}, ErrAmountTooSmall
	}
	return Swap{Quote: quote, Writes: writes}, nil
}

// SellExactBase is NightlyPool._sellExactBase without a rebalance, as swapExactIn and
// quoteSwapExactIn run it: it sells exactly baseIn for quote, walking down from the position at
// the oracle's bid spread. blockTimestamp and the returned Swap are as in BuyWithExactQuote.
func (p *Pool) SellExactBase(baseIn *uint256.Int, oracle OraclePrice, blockTimestamp uint64) (Swap, error) {
	if baseIn.IsZero() {
		return Swap{}, ErrZeroSwapAmount
	}
	mid, err := p.walkPrice(oracle, blockTimestamp)
	if err != nil {
		return Swap{}, err
	}
	quote := Quote{AmountIn: *baseIn, EndPosition: p.Position}
	remaining := new(uint256.Int).Set(baseIn)
	var writes []LevelWrite
	for {
		step, write, err := p.sellIntoLevel(quote.EndPosition, remaining, mid, oracle.BidSpread)
		if err != nil {
			return Swap{}, err
		}
		if write != nil {
			writes = append(writes, *write)
		}
		if remaining, err = Sub(remaining, &step.BaseAmount); err != nil {
			return Swap{}, err
		}
		amountOut, err := Add(&quote.AmountOut, &step.QuoteAmount)
		if err != nil {
			return Swap{}, err
		}
		quote.AmountOut = *amountOut
		quote.EndPosition.Fraction = step.End
		// An exact refill ends at the lower edge. Only more offered base moves to the next level down.
		if remaining.IsZero() {
			break
		}
		if quote.EndPosition.LevelID == p.LowestLevelID {
			return Swap{}, fmt.Errorf("%w: %s left", ErrInsufficientLiquidity, remaining.Dec())
		}
		quote.EndPosition = Position{LevelID: quote.EndPosition.LevelID - 1, Fraction: Upper}
	}
	// Quote outputs round down, so a tiny sale can receive nothing.
	if quote.AmountOut.IsZero() {
		return Swap{}, ErrAmountTooSmall
	}
	return Swap{Quote: quote, Writes: writes}, nil
}

// walkPrice is NightlyPool._oraclePrice's age check and the mid price that a walk prices from. The
// price is stale when its time plus the maximum price age is before the block's time.
func (p *Pool) walkPrice(oracle OraclePrice, blockTimestamp uint64) (Price, error) {
	if oracle.TimestampMs+uint64(p.MaxPriceAgeMs) < blockTimestamp*1000 {
		return Price{}, fmt.Errorf("%w: price time %d ms, maximum age %d ms", ErrStaleOraclePrice,
			oracle.TimestampMs, p.MaxPriceAgeMs)
	}
	return PriceFromOracle(oracle.MidPrice, p.BaseDecimals, p.QuoteDecimals)
}

// buyFromLevel is NightlyPool._buyWithExactQuoteFromLevel and _recordBuy: it prices base leaving
// one level and checks that the level can record the incoming quote. A level without base returns
// an empty step without pricing it. The quote paid stays in the level that sold the base.
func (p *Pool) buyFromLevel(position Position, offeredQuote *uint256.Int, mid Price,
	oracleAskSpread Spread) (Step, *LevelWrite, error) {
	level, err := p.level(position.LevelID)
	if err != nil {
		return Step{}, nil, err
	}
	liquidity := level.Liquidity
	if liquidity.BaseAmount.IsZero() {
		return Step{End: position.Fraction}, nil, nil
	}
	ask, err := AskRange(level.Config, mid, oracleAskSpread)
	if err != nil {
		return Step{}, nil, err
	}
	step, err := BuyWithExactQuote(&liquidity.BaseAmount, position.Fraction, offeredQuote, ask)
	if err != nil {
		return Step{}, nil, err
	}
	quoteAmount, overflow := new(uint256.Int).AddOverflow(&liquidity.QuoteAmount, &step.QuoteAmount)
	if overflow {
		return Step{}, nil, fmt.Errorf("%w: level %d", ErrLiquidityOverflow, position.LevelID)
	}
	write := &LevelWrite{LevelID: position.LevelID}
	write.Liquidity.BaseAmount.Sub(&liquidity.BaseAmount, &step.BaseAmount)
	write.Liquidity.QuoteAmount = *quoteAmount
	return step, write, nil
}

// sellIntoLevel is NightlyPool._sellExactBaseIntoLevel and _recordSell: it prices base entering
// one level and checks that the level can record that base. A level without quote returns an
// empty step without pricing it. The base sold stays in the level that paid the quote.
func (p *Pool) sellIntoLevel(position Position, offeredBase *uint256.Int, mid Price,
	oracleBidSpread Spread) (Step, *LevelWrite, error) {
	level, err := p.level(position.LevelID)
	if err != nil {
		return Step{}, nil, err
	}
	liquidity := level.Liquidity
	if liquidity.QuoteAmount.IsZero() {
		return Step{End: position.Fraction}, nil, nil
	}
	bid, err := BidRange(level.Config, mid, oracleBidSpread)
	if err != nil {
		return Step{}, nil, err
	}
	step, err := SellExactBase(&liquidity.QuoteAmount, position.Fraction, offeredBase, bid)
	if err != nil {
		return Step{}, nil, err
	}
	baseAmount, overflow := new(uint256.Int).AddOverflow(&liquidity.BaseAmount, &step.BaseAmount)
	if overflow {
		return Step{}, nil, fmt.Errorf("%w: level %d", ErrLiquidityOverflow, position.LevelID)
	}
	write := &LevelWrite{LevelID: position.LevelID}
	write.Liquidity.BaseAmount = *baseAmount
	write.Liquidity.QuoteAmount.Sub(&liquidity.QuoteAmount, &step.QuoteAmount)
	return step, write, nil
}

// Apply writes a swap's level balances and its end position into the pool, as the swap's storage
// writes. It checks every level first, so a swap with a level outside the pool writes nothing.
func (p *Pool) Apply(swap Swap) error {
	for _, write := range swap.Writes {
		if _, err := p.level(write.LevelID); err != nil {
			return err
		}
	}
	if _, err := p.level(swap.Quote.EndPosition.LevelID); err != nil {
		return err
	}
	for _, write := range swap.Writes {
		p.Levels[int(write.LevelID)-int(p.LowestLevelID)].Liquidity = write.Liquidity
	}
	p.Position = swap.Quote.EndPosition
	return nil
}

// level returns the level with ID levelID. An ID outside the pool's levels is
// ErrInvalidPriceLevel, as the pool's level getters revert.
func (p *Pool) level(levelID int16) (*Level, error) {
	index := int(levelID) - int(p.LowestLevelID)
	if index < 0 || index >= len(p.Levels) {
		return nil, fmt.Errorf("%w: %d", ErrInvalidPriceLevel, levelID)
	}
	return &p.Levels[index], nil
}

func (p *Pool) highestLevelID() int16 {
	return int16(int(p.LowestLevelID) + len(p.Levels) - 1)
}
