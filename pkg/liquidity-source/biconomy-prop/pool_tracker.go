package biconomyprop

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	feed         *resty.Client // nil: no feed configured
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	t := &PoolTracker{config: cfg, ethrpcClient: ethrpcClient}
	if cfg.HTTP != nil && cfg.HTTP.BaseURL != "" {
		t.feed = resty.New().SetBaseURL(cfg.HTTP.BaseURL).SetHeaders(cfg.HTTP.Headers).SetTimeout(3 * time.Second)
	}
	return t, nil
}

// rawBoard is PropAMMVenue.board()'s return tuple, decoded by output name.
type rawBoard struct {
	Sizes     []*big.Int
	Prices    []*big.Int
	Filled    *big.Int
	Remaining *big.Int
	ExpiresAt *big.Int
}

// GetNewPoolState reads the board feed, falling back to on-chain reads when the feed fails.
func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	if len(p.Tokens) != 2 {
		return p, ErrInvalidToken
	}
	token0, token1 := common.HexToAddress(p.Tokens[0].Address), common.HexToAddress(p.Tokens[1].Address)

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	members, feeBps, blockNumber, err := t.fetchFeed(ctx, token0, token1, staticExtra.Venue)
	if err != nil {
		logger.WithFields(logger.Fields{"dexId": t.config.DexID, "pool": p.Address}).
			Warnf("biconomy-prop: board feed unusable, reading on-chain: %v", err)
		if members, feeBps, blockNumber, err = t.fetchOnchain(ctx, token0, token1, staticExtra.Venue); err != nil {
			return p, err
		}
	}

	extra, err := json.Marshal(Extra{Members: members, FeeBps: feeBps})
	if err != nil {
		return p, err
	}

	// Each reserve is what the direction paying that token out can deliver, before the protocol fee.
	now := time.Now().Unix()
	reserve1 := deliverable(members, 0, uint64(now)) // token0 -> token1 pays out token1
	reserve0 := deliverable(members, 1, uint64(now)) // token1 -> token0 pays out token0

	p.Extra = string(extra)
	p.Reserves = entity.PoolReserves{reserve0.Dec(), reserve1.Dec()}
	p.BlockNumber = blockNumber
	p.Timestamp = now
	return p, nil
}

// fetchOnchain reads the venue's makers and protocol fee, then every maker's board for both
// directions of the pair, pinned to the same block.
func (t *PoolTracker) fetchOnchain(ctx context.Context, token0, token1 common.Address, venue string) (
	[]MemberExtra, uint64, uint64, error) {
	var (
		makerAddrs []common.Address
		feeBps     uint16
	)
	makersResp, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: venueABI, Target: venue, Method: methodMakers}, []any{&makerAddrs}).
		AddCall(&ethrpc.Call{ABI: venueABI, Target: venue, Method: methodFeeBps}, []any{&feeBps}).
		Aggregate()
	if err != nil {
		return nil, 0, 0, err
	}

	boards := make([]rawBoard, 2*len(makerAddrs))
	if len(makerAddrs) > 0 {
		req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(makersResp.BlockNumber)
		for i, m := range makerAddrs {
			req.AddCall(&ethrpc.Call{ABI: venueABI, Target: venue, Method: methodBoard, Params: []any{m, token0, token1}},
				[]any{&boards[2*i]})
			req.AddCall(&ethrpc.Call{ABI: venueABI, Target: venue, Method: methodBoard, Params: []any{m, token1, token0}},
				[]any{&boards[2*i+1]})
		}
		if _, err := req.Aggregate(); err != nil {
			return nil, 0, 0, err
		}
	}

	members := make([]MemberExtra, len(makerAddrs))
	for i, m := range makerAddrs {
		members[i] = MemberExtra{
			Maker: hexutil.Encode(m[:]),
			Dir0:  toBoard(boards[2*i]),
			Dir1:  toBoard(boards[2*i+1]),
		}
	}
	return members, uint64(feeBps), makersResp.BlockNumber.Uint64(), nil
}

func toBoard(rb rawBoard) Board {
	b := Board{
		Sizes:     make([]*uint256.Int, len(rb.Sizes)),
		Prices:    make([]*uint256.Int, len(rb.Prices)),
		Filled:    big256.FromBig(rb.Filled),
		Remaining: big256.FromBig(rb.Remaining),
		ExpiresAt: 0,
	}
	if rb.ExpiresAt != nil {
		b.ExpiresAt = rb.ExpiresAt.Uint64()
	}
	for i, s := range rb.Sizes {
		b.Sizes[i] = big256.FromBig(s)
	}
	for i, pr := range rb.Prices {
		b.Prices[i] = big256.FromBig(pr)
	}
	return b
}

// deliverable sums the output of filling every live board of a direction up to its limit.
func deliverable(members []MemberExtra, dirIndex int, now uint64) *uint256.Int {
	total := new(uint256.Int)
	var limit, cursor, out, take uint256.Int
	for i := range members {
		board := &members[i].Dir0
		if dirIndex == 1 {
			board = &members[i].Dir1
		}
		if !boardLive(board, now) {
			continue
		}
		fillLimit(&limit, board)
		cursor.Set(board.Filled)
		for j := range board.Sizes {
			if !cursor.Lt(&limit) {
				break
			}
			top := board.Sizes[j]
			if !cursor.Lt(top) {
				continue
			}
			if top.Gt(&limit) {
				top = &limit
			}
			take.Sub(top, &cursor)
			segmentOut(&out, &take, board.Prices[j])
			total.Add(total, &out)
			cursor.Set(top)
		}
	}
	return total
}
