package biconomyprop

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
)

// feedResponse is GET /venue/boards. Numbers are decimal strings, same units as board().
type feedResponse struct {
	Venue      string      `json:"venue"`
	StateBlock string      `json:"stateBlock"`
	FeeBps     uint64      `json:"feeBps"`
	Makers     []string    `json:"makers"`
	BuiltAtMs  int64       `json:"builtAtMs"`
	Boards     []feedBoard `json:"boards"`
}

type feedBoard struct {
	MM        string   `json:"mm"`
	TokenIn   string   `json:"tokenIn"`
	TokenOut  string   `json:"tokenOut"`
	Committed bool     `json:"committed"`
	Sizes     []string `json:"sizes"`
	Prices    []string `json:"prices"`
	Filled    string   `json:"filled"`
	Remaining string   `json:"remaining"`
	ExpiresAt string   `json:"expiresAt"`
}

// fetchFeed builds the members from the REST feed. Only committed boards count: the venue fills
// committed state only. The pool block is stateBlock, the block the boards were read at.
func (t *PoolTracker) fetchFeed(ctx context.Context, token0, token1 common.Address, venue string) (
	[]MemberExtra, uint64, uint64, error) {
	if t.feed == nil {
		return nil, 0, 0, fmt.Errorf("%w: no feed configured", ErrFeedInvalid)
	}
	var resp feedResponse
	r, err := t.feed.R().SetContext(ctx).
		SetQueryParam("chainId", strconv.Itoa(int(t.config.ChainID))).Get("/venue/boards")
	if err != nil {
		return nil, 0, 0, err
	}
	if r.IsError() {
		return nil, 0, 0, fmt.Errorf("%w: http %d", ErrFeedInvalid, r.StatusCode())
	}
	if err := json.Unmarshal(r.Body(), &resp); err != nil {
		return nil, 0, 0, fmt.Errorf("%w: %v", ErrFeedInvalid, err)
	}
	return buildFeedMembers(&resp, token0, token1, venue, t.maxAgeMs(), time.Now().UnixMilli())
}

func (t *PoolTracker) maxAgeMs() int64 {
	if t.config.MaxAgeMs > 0 {
		return t.config.MaxAgeMs
	}
	return defaultFeedMaxAgeMs
}

func buildFeedMembers(resp *feedResponse, token0, token1 common.Address, venue string, maxAgeMs, nowMs int64) (
	[]MemberExtra, uint64, uint64, error) {
	if !strings.EqualFold(resp.Venue, venue) {
		return nil, 0, 0, fmt.Errorf("%w: venue %q", ErrFeedInvalid, resp.Venue)
	}
	block, err := strconv.ParseUint(resp.StateBlock, 10, 64)
	if err != nil || block == 0 || resp.BuiltAtMs == 0 || resp.Makers == nil {
		return nil, 0, 0, fmt.Errorf("%w: stateBlock %q", ErrFeedInvalid, resp.StateBlock)
	}
	if age := nowMs - resp.BuiltAtMs; age > maxAgeMs || age < -maxAgeMs {
		return nil, 0, 0, fmt.Errorf("%w: age %dms > %dms", ErrFeedStale, age, maxAgeMs)
	}
	if len(resp.Makers) > maxMakers {
		return nil, 0, 0, ErrTooManyMakers
	}

	// key: maker, tokenIn (lowercase) -> committed board of the pair
	boards := make(map[[2]string]*feedBoard, len(resp.Boards))
	for i := range resp.Boards {
		b := &resp.Boards[i]
		if !b.Committed {
			continue
		}
		tin, tout := strings.ToLower(b.TokenIn), strings.ToLower(b.TokenOut)
		if (tin == hexutil.Encode(token0[:]) && tout == hexutil.Encode(token1[:])) ||
			(tin == hexutil.Encode(token1[:]) && tout == hexutil.Encode(token0[:])) {
			boards[[2]string{strings.ToLower(b.MM), tin}] = b
		}
	}

	members := make([]MemberExtra, len(resp.Makers))
	for i, m := range resp.Makers {
		maker := strings.ToLower(m)
		var err0, err1 error
		members[i].Maker = maker
		members[i].Dir0, err0 = feedToBoard(boards[[2]string{maker, hexutil.Encode(token0[:])}])
		members[i].Dir1, err1 = feedToBoard(boards[[2]string{maker, hexutil.Encode(token1[:])}])
		if err0 != nil || err1 != nil {
			return nil, 0, 0, fmt.Errorf("%w: maker %s: %v %v", ErrFeedInvalid, maker, err0, err1)
		}
	}
	return members, resp.FeeBps, block, nil
}

// feedToBoard converts a feed board; nil (missing direction) becomes a dark board.
func feedToBoard(fb *feedBoard) (Board, error) {
	b := Board{Sizes: []*uint256.Int{}, Prices: []*uint256.Int{}, Filled: new(uint256.Int), Remaining: new(uint256.Int)}
	if fb == nil {
		return b, nil
	}
	if len(fb.Sizes) != len(fb.Prices) {
		return b, fmt.Errorf("sizes/prices length mismatch")
	}
	var err error
	if b.Filled, err = uint256.FromDecimal(fb.Filled); err != nil {
		return b, err
	}
	if b.Remaining, err = uint256.FromDecimal(fb.Remaining); err != nil {
		return b, err
	}
	if b.ExpiresAt, err = strconv.ParseUint(fb.ExpiresAt, 10, 64); err != nil {
		return b, err
	}
	for i := range fb.Sizes {
		s, err := uint256.FromDecimal(fb.Sizes[i])
		if err != nil {
			return b, err
		}
		p, err := uint256.FromDecimal(fb.Prices[i])
		if err != nil {
			return b, err
		}
		b.Sizes, b.Prices = append(b.Sizes, s), append(b.Prices, p)
	}
	return b, nil
}
