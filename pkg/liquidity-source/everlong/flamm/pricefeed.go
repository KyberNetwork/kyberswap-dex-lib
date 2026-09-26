package everlongflamm

import (
	"errors"

	"github.com/holiman/uint256"
)

// PriceFeed (src/core/PriceFeed.sol @ c104 80abd43, 0xbED275459578C87a63F2f50A0b077C720e838816 on Base): the
// checked USD quote of each registered token from its Chainlink USD aggregator behind the L2 sequencer uptime
// feed, and the cross of two quotes. Only the paths the swap and leverage routes reach are ported: cross,
// peekCross, usd, peekUsd and pegOk. The state is each aggregator's latestRoundData as read at the snapshot plus
// the immutable per-token config; every check is evaluated at the call's timestamp, so a snapshot ages exactly
// as the chain does (StalePrice past the heartbeat, SequencerGrace inside the grace) until a new round lands.

// errFeedReverted stands for a latestRoundData call that reverted; the revert data (the aggregator's own) is
// bubbled on-chain and not modelled.
var errFeedReverted = errors.New("everlong-flamm: latestRoundData reverted")

// feedMaxPriceWad is PriceFeed.MAX_PRICE_WAD = 1 << 200.
var feedMaxPriceWad = new(uint256.Int).Lsh(uOne, 200)

// feedRound is one aggregator's latestRoundData (roundId, answer, startedAt, updatedAt); Ok is false when the
// call reverted. Answer is the int256 as its two's-complement word.
type feedRound struct {
	Ok        bool        `json:"ok"`
	RoundId   uint256.Int `json:"roundId"`
	Answer    uint256.Int `json:"answer"`
	StartedAt uint256.Int `json:"startedAt"`
	UpdatedAt uint256.Int `json:"updatedAt"`
}

// feedToken is PriceFeed.Token (config(token)) with its aggregator's round. Known is false for a token the
// feed never registered (UnknownToken).
type feedToken struct {
	Known      bool        `json:"known"`
	Heartbeat  uint256.Int `json:"heartbeat"`
	Scale      uint256.Int `json:"scale"` // 10**(18 - feed decimals)
	Unit       uint256.Int `json:"unit"`  // 10**token decimals
	PegBandWad uint256.Int `json:"pegBandWad"`
	Round      feedRound   `json:"round"`
}

// priceFeedState is the PriceFeed as the pool's paths read it: the sequencer feed (absent when
// SEQUENCER_FEED == 0) with SEQUENCER_GRACE, the pool asset's token and each loan asset's, in loan order.
type priceFeedState struct {
	HasSequencer   bool        `json:"hasSequencer"`
	SequencerGrace uint256.Int `json:"sequencerGrace"`
	Sequencer      feedRound   `json:"sequencer"`
	Asset          feedToken   `json:"asset"`
	Loans          []feedToken `json:"loans"`
}

func (f priceFeedState) clone() priceFeedState {
	f.Loans = append([]feedToken(nil), f.Loans...)
	return f
}

// requireSequencer is PriceFeed._requireSequencer (:157): a non-zero answer or a zero startedAt is down, and
// `block.timestamp - startedAt <= SEQUENCER_GRACE` (a checked subtraction) is inside the grace.
func (f *priceFeedState) requireSequencer(now uint64) error {
	if !f.HasSequencer {
		return nil
	}
	r := &f.Sequencer
	if !r.Ok {
		return errFeedReverted
	}
	if !r.Answer.IsZero() || r.StartedAt.IsZero() {
		return ErrSequencerDown
	}
	var up uint256.Int
	up.SetUint64(now)
	if up.Lt(&r.StartedAt) {
		return errPanicArithmetic
	}
	if !up.Sub(&up, &r.StartedAt).Gt(&f.SequencerGrace) {
		return ErrSequencerGrace
	}
	return nil
}

// feedRead is PriceFeed._read (:164): the round must be a positive answer with a set, non-future timestamp no
// older than the heartbeat; usdWad = answer * scale (checked). Returns the round's updatedAt.
func feedRead(t *feedToken, now uint64) (uint256.Int, uint64, error) {
	r := &t.Round
	if !r.Ok {
		return uint256.Int{}, 0, errFeedReverted
	}
	var nowU uint256.Int
	nowU.SetUint64(now)
	if r.RoundId.IsZero() || r.Answer.IsZero() || r.Answer.Sign() < 0 || r.UpdatedAt.IsZero() || r.UpdatedAt.Gt(&nowU) {
		return uint256.Int{}, 0, ErrInvalidPrice
	}
	if nowU.Sub(&nowU, &r.UpdatedAt).Gt(&t.Heartbeat) {
		return uint256.Int{}, 0, ErrStalePrice
	}
	usd, err := gateMul(&r.Answer, &t.Scale)
	return usd, r.UpdatedAt.Uint64(), err
}

// usd is PriceFeed.usd (:68): the sequencer first, then the registration, then the round.
func (f *priceFeedState) usd(t *feedToken, now uint64) (uint256.Int, uint64, error) {
	if err := f.requireSequencer(now); err != nil {
		return uint256.Int{}, 0, err
	}
	if !t.Known {
		return uint256.Int{}, 0, ErrUnknownToken
	}
	return feedRead(t, now)
}

// peekUsd is PriceFeed.peekUsd (:74): any revert of usd reads as (false, 0, 0).
func (f *priceFeedState) peekUsd(t *feedToken, now uint64) (bool, uint256.Int, uint64) {
	v, ts, err := f.usd(t, now)
	if err != nil {
		return false, uint256.Int{}, 0
	}
	return true, v, ts
}

// cross is PriceFeed.cross (:108): priceWad = mulDiv(baseUsd, WAD, quoteUsd) / base.unit, N18 per base unit,
// which must lie in (0, 2^200); observedAt is the older of the two rounds.
func (f *priceFeedState) cross(base, quote *feedToken, now uint64) (uint256.Int, uint64, error) {
	var p uint256.Int
	if err := f.requireSequencer(now); err != nil {
		return p, 0, err
	}
	if !base.Known {
		return p, 0, ErrUnknownToken
	}
	baseUsd, baseTs, err := feedRead(base, now)
	if err != nil {
		return p, 0, err
	}
	if !quote.Known {
		return p, 0, ErrUnknownToken
	}
	quoteUsd, quoteTs, err := feedRead(quote, now)
	if err != nil {
		return p, 0, err
	}
	if p, err = mmMulDivOZ(&baseUsd, uWad, &quoteUsd); err != nil {
		return p, 0, err
	}
	if base.Unit.IsZero() {
		return p, 0, errPanicDivZero
	}
	if p.Div(&p, &base.Unit).IsZero() || !p.Lt(feedMaxPriceWad) {
		return uint256.Int{}, 0, ErrInvalidPrice
	}
	return p, min(baseTs, quoteTs), nil
}

// peekCross is PriceFeed.peekCross (:119): any revert of cross reads as (false, 0, 0).
func (f *priceFeedState) peekCross(base, quote *feedToken, now uint64) (bool, uint256.Int, uint64) {
	p, ts, err := f.cross(base, quote, now)
	if err != nil {
		return false, uint256.Int{}, 0
	}
	return true, p, ts
}

// pegOk is PriceFeed.pegOk (:83): an unregistered token reverts UnknownToken (outside the try); a zero band is
// always ok; otherwise a reverting usd is not ok, and |usd - WAD| must be within the band.
func (f *priceFeedState) pegOk(t *feedToken, now uint64) (bool, error) {
	if !t.Known {
		return false, ErrUnknownToken
	}
	if t.PegBandWad.IsZero() {
		return true, nil
	}
	v, _, err := f.usd(t, now)
	if err != nil {
		return false, nil
	}
	var dev uint256.Int
	if v.Gt(uWad) {
		dev.Sub(&v, uWad)
	} else {
		dev.Sub(uWad, &v)
	}
	return !dev.Gt(&t.PegBandWad), nil
}
