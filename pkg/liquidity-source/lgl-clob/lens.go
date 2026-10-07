package lglclob

import (
	"context"
	"encoding/binary"
	"slices"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

const (
	lensTag    = "Lens"
	lensHeader = 33 // the tag, the block, whether there is a market maker, and four quotes
	lensLevel  = 25 // a price in 9 bytes and shares in 16
)

// quote is one size Lens had a pool's market maker quote, and what it
// measured.
type quote struct {
	buy    bool
	gas    int64
	levels int64
}

// lensState is what Lens read of a pool at block. hasMaker is whether the pool
// has a market maker, and quotes are a sell at one level, a sell at two, a buy
// at one and a buy at two.
type lensState struct {
	block    uint64
	hasMaker bool
	quotes   [4]quote
	book     OrderBook
}

// readState reads pool's book in one eth_call through Lens, a deployless
// helper, and when measure is set what its market maker's quote costs. Lens reads the
// book through helper.
func readState(ctx context.Context, client *ethrpc.Client, pool, helper common.Address,
	measure bool) (*lensState, error) {
	args, err := lensABI.Pack("", pool, helper, bMaxPriceLevels, measure)
	if err != nil {
		return nil, err
	}
	data, err := eth.DeploylessCall(ctx, client.GetETHClient().Client(), slices.Concat(lensBytecode, args), nil,
		nil)
	if err != nil {
		return nil, err
	}
	return decodeState(data)
}

// decodeState reads Lens's packed revert, laid out as abis/Lens.sol describes.
func decodeState(data []byte) (*lensState, error) {
	if len(data) == 0 {
		return nil, ErrBookUnread
	}
	if len(data) < lensHeader+2 || string(data[:4]) != lensTag {
		return nil, ErrUnexpectedLensRevert
	}
	state := &lensState{
		block:    binary.BigEndian.Uint64(data[4:12]),
		hasMaker: data[12] == 1,
	}
	for i := range state.quotes {
		at := 13 + 5*i
		state.quotes[i] = quote{
			buy:    i >= 2,
			gas:    int64(binary.BigEndian.Uint32(data[at : at+4])),
			levels: int64(data[at+4]),
		}
	}

	rest := data[lensHeader:]
	side := func() (OrderBookLevels, bool) {
		if len(rest) == 0 {
			return OrderBookLevels{}, false
		}
		n := int(rest[0])
		if len(rest) < 1+n*lensLevel {
			return OrderBookLevels{}, false
		}
		levels := OrderBookLevels{
			ArrayPrices: make([]*uint256.Int, n),
			ArrayShares: make([]*uint256.Int, n),
		}
		for i := range n {
			at := 1 + i*lensLevel
			levels.ArrayPrices[i] = new(uint256.Int).SetBytes(rest[at : at+9])
			levels.ArrayShares[i] = new(uint256.Int).SetBytes(rest[at+9 : at+lensLevel])
		}
		rest = rest[1+n*lensLevel:]
		return levels, true
	}
	var ok bool
	if state.book.Bids, ok = side(); !ok {
		return nil, ErrUnexpectedLensRevert
	}
	if state.book.Asks, ok = side(); !ok || len(rest) != 0 {
		return nil, ErrUnexpectedLensRevert
	}
	return state, nil
}

// makerQuoteGas is the gas of the market maker's quote s measured, nil for a
// pool without a market maker.
func (s *lensState) makerQuoteGas() *MakerQuoteGas {
	if !s.hasMaker {
		return nil
	}
	return &MakerQuoteGas{Bids: fitLevelGas(s.quotes[:], false), Asks: fitLevelGas(s.quotes[:], true)}
}

// measureDue reports whether the market maker's quote is to be measured again
// at now, as last was measured makerQuoteRefresh ago or never, or a side last
// could not quote has orders, after makerQuoteRetry. A side without orders, as a new pool
// may have, is not quoted, and is measured soon after orders come.
func measureDue(last Extra, now time.Time) bool {
	since := now.Sub(time.Unix(last.MeasuredAt, 0))
	if since >= makerQuoteRefresh {
		return true
	}
	measured := last.MakerQuoteGas
	unquoted := measured != nil && (measured.Bids == nil && len(last.Bids.ArrayShares) > 0 ||
		measured.Asks == nil && len(last.Asks.ArrayShares) > 0)
	return unquoted && since >= makerQuoteRetry
}

// fitLevelGas fits a side's market maker quote from its quotes. Two quotes
// filling a different number of levels give what each level more adds, and a
// single one takes unmeasuredMakerQuote's. A quote that failed or quoted nothing is left out,
// and a side without any is nil.
func fitLevelGas(quotes []quote, buy bool) *LevelGas {
	var fit []quote
	for _, q := range quotes {
		if q.buy == buy && q.gas > 0 && q.levels > 0 {
			fit = append(fit, q)
		}
	}
	if len(fit) == 0 {
		return nil
	}
	slices.SortFunc(fit, func(a, b quote) int { return int(a.levels - b.levels) })

	low, high := fit[0], fit[len(fit)-1]
	next := unmeasuredMakerQuote.Next
	if high.levels > low.levels {
		next = max(0, (high.gas-low.gas)/(high.levels-low.levels))
	}
	return &LevelGas{First: max(0, low.gas-next*(low.levels-1)), Next: next}
}
