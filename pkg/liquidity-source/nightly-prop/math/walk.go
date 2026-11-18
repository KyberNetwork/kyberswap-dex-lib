package math

import (
	"fmt"

	"github.com/holiman/uint256"
)

// Liquidity holds a level's balances in native token units.
type Liquidity struct {
	BaseAmount  uint256.Int `json:"baseAmount"`
	QuoteAmount uint256.Int `json:"quoteAmount"`
}

// Level is one price level: its configuration and its balances.
type Level struct {
	Config    PriceLevelConfig `json:"config"`
	Liquidity Liquidity        `json:"liquidity"`
}

// Position identifies the current level and the fraction within it.
type Position struct {
	LevelID  int16    `json:"levelId"`
	Fraction Fraction `json:"fraction"`
}

// OraclePrice stores the midpoint with eight decimals, spreads in millionths, and a timestamp in milliseconds.
type OraclePrice struct {
	MidPrice    uint64 `json:"midPrice"`
	BidSpread   Spread `json:"bidSpread"`
	AskSpread   Spread `json:"askSpread"`
	TimestampMs uint64 `json:"timestampMs"`
}

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

// BuyWithExactQuote buys base by walking up the asks from the current position.
// blockTimestamp is in seconds. Apply commits the returned level changes.
func (p *Pool) BuyWithExactQuote(quoteIn *uint256.Int, oracle OraclePrice, blockTimestamp uint64) (Swap, error) {
	if quoteIn.IsZero() {
		return Swap{}, ErrZeroSwapAmount
	}
	mid, err := p.walkPrice(oracle, blockTimestamp)
	if err != nil {
		return Swap{}, err
	}
	quote := Quote{AmountIn: *quoteIn, EndPosition: p.Position}
	remaining := *quoteIn
	var writes []LevelWrite
	for {
		step, write, wrote, err := p.buyFromLevel(quote.EndPosition, &remaining, &mid, oracle.AskSpread)
		if err != nil {
			return Swap{}, err
		}
		if wrote {
			writes = append(writes, write)
		}
		if remaining, err = Sub(&remaining, &step.QuoteAmount); err != nil {
			return Swap{}, err
		}
		if quote.AmountOut, err = Add(&quote.AmountOut, &step.BaseAmount); err != nil {
			return Swap{}, err
		}
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

// SellExactBase sells base by walking down the bids from the current position.
// blockTimestamp is in seconds. Apply commits the returned level changes.
func (p *Pool) SellExactBase(baseIn *uint256.Int, oracle OraclePrice, blockTimestamp uint64) (Swap, error) {
	if baseIn.IsZero() {
		return Swap{}, ErrZeroSwapAmount
	}
	mid, err := p.walkPrice(oracle, blockTimestamp)
	if err != nil {
		return Swap{}, err
	}
	quote := Quote{AmountIn: *baseIn, EndPosition: p.Position}
	remaining := *baseIn
	var writes []LevelWrite
	for {
		step, write, wrote, err := p.sellIntoLevel(quote.EndPosition, &remaining, &mid, oracle.BidSpread)
		if err != nil {
			return Swap{}, err
		}
		if wrote {
			writes = append(writes, write)
		}
		if remaining, err = Sub(&remaining, &step.BaseAmount); err != nil {
			return Swap{}, err
		}
		if quote.AmountOut, err = Add(&quote.AmountOut, &step.QuoteAmount); err != nil {
			return Swap{}, err
		}
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

// A price is valid through its timestamp plus the maximum age.
func (p *Pool) walkPrice(oracle OraclePrice, blockTimestamp uint64) (Price, error) {
	if oracle.TimestampMs+uint64(p.MaxPriceAgeMs) < blockTimestamp*1000 {
		return Price{}, fmt.Errorf("%w: price time %d ms, maximum age %d ms", ErrStaleOraclePrice,
			oracle.TimestampMs, p.MaxPriceAgeMs)
	}
	return PriceFromOracle(oracle.MidPrice, p.BaseDecimals, p.QuoteDecimals)
}

// buyFromLevel skips empty asks and credits the incoming quote to the level that sold the base.
func (p *Pool) buyFromLevel(position Position, offeredQuote *uint256.Int, mid *Price,
	oracleAskSpread Spread) (step Step, write LevelWrite, wrote bool, err error) {
	level, err := p.level(position.LevelID)
	if err != nil {
		return step, write, false, err
	}
	liquidity := &level.Liquidity
	if liquidity.BaseAmount.IsZero() {
		return Step{End: position.Fraction}, write, false, nil
	}
	ask, err := AskRange(level.Config, *mid, oracleAskSpread)
	if err != nil {
		return step, write, false, err
	}
	if step, err = BuyWithExactQuote(&liquidity.BaseAmount, position.Fraction, offeredQuote, ask); err != nil {
		return step, write, false, err
	}
	write.LevelID = position.LevelID
	if _, overflow := write.Liquidity.QuoteAmount.AddOverflow(&liquidity.QuoteAmount, &step.QuoteAmount); overflow {
		return step, write, false, fmt.Errorf("%w: level %d", ErrLiquidityOverflow, position.LevelID)
	}
	write.Liquidity.BaseAmount.Sub(&liquidity.BaseAmount, &step.BaseAmount)
	return step, write, true, nil
}

// sellIntoLevel skips empty bids and credits the incoming base to the level that paid the quote.
func (p *Pool) sellIntoLevel(position Position, offeredBase *uint256.Int, mid *Price,
	oracleBidSpread Spread) (step Step, write LevelWrite, wrote bool, err error) {
	level, err := p.level(position.LevelID)
	if err != nil {
		return step, write, false, err
	}
	liquidity := &level.Liquidity
	if liquidity.QuoteAmount.IsZero() {
		return Step{End: position.Fraction}, write, false, nil
	}
	bid, err := BidRange(level.Config, *mid, oracleBidSpread)
	if err != nil {
		return step, write, false, err
	}
	if step, err = SellExactBase(&liquidity.QuoteAmount, position.Fraction, offeredBase, bid); err != nil {
		return step, write, false, err
	}
	write.LevelID = position.LevelID
	if _, overflow := write.Liquidity.BaseAmount.AddOverflow(&liquidity.BaseAmount, &step.BaseAmount); overflow {
		return step, write, false, fmt.Errorf("%w: level %d", ErrLiquidityOverflow, position.LevelID)
	}
	write.Liquidity.QuoteAmount.Sub(&liquidity.QuoteAmount, &step.QuoteAmount)
	return step, write, true, nil
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
