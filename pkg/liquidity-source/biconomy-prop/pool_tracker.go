package biconomyprop

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: cfg, ethrpcClient: ethrpcClient}, nil
}

// rawBoard is PropAMMVenue.board()'s return tuple, decoded by output name.
type rawBoard struct {
	Sizes     []*big.Int
	Prices    []*big.Int
	Filled    *big.Int
	Remaining *big.Int
	ExpiresAt *big.Int
}

// GetNewPoolState reads the venue's makers and protocol fee, then every maker's board for both
// directions of the pair, pinned to the same block.
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

	var (
		makerAddrs []common.Address
		feeBps     uint16
	)
	makersResp, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: venueABI, Target: staticExtra.Venue, Method: methodMakers}, []any{&makerAddrs}).
		AddCall(&ethrpc.Call{ABI: venueABI, Target: staticExtra.Venue, Method: methodFeeBps}, []any{&feeBps}).
		Aggregate()
	if err != nil {
		return p, err
	}

	boards := make([]rawBoard, 2*len(makerAddrs))
	if len(makerAddrs) > 0 {
		req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(makersResp.BlockNumber)
		for i, m := range makerAddrs {
			req.AddCall(&ethrpc.Call{ABI: venueABI, Target: staticExtra.Venue, Method: methodBoard, Params: []any{m, token0, token1}},
				[]any{&boards[2*i]})
			req.AddCall(&ethrpc.Call{ABI: venueABI, Target: staticExtra.Venue, Method: methodBoard, Params: []any{m, token1, token0}},
				[]any{&boards[2*i+1]})
		}
		if _, err := req.Aggregate(); err != nil {
			return p, err
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

	extra, err := json.Marshal(Extra{Members: members, FeeBps: uint64(feeBps)})
	if err != nil {
		return p, err
	}

	// Each reserve is what the direction paying that token out can deliver, before the protocol fee.
	now := uint64(time.Now().Unix())
	reserve1 := deliverable(members, 0, now) // token0 -> token1 pays out token1
	reserve0 := deliverable(members, 1, now) // token1 -> token0 pays out token0

	p.Extra = string(extra)
	p.Reserves = entity.PoolReserves{reserve0.Dec(), reserve1.Dec()}
	p.BlockNumber = makersResp.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()
	return p, nil
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
	for i := range members {
		board := &members[i].Dir0
		if dirIndex == 1 {
			board = &members[i].Dir1
		}
		if !boardLive(board, now) {
			continue
		}
		limit := fillLimit(board)
		cursor := board.Filled.Clone()
		var out, take uint256.Int
		for j := range board.Sizes {
			if !cursor.Lt(limit) {
				break
			}
			top := board.Sizes[j]
			if !cursor.Lt(top) {
				continue
			}
			if top.Gt(limit) {
				top = limit
			}
			take.Sub(top, cursor)
			segmentOut(&out, &take, board.Prices[j])
			total.Add(total, &out)
			cursor.Set(top)
		}
	}
	return total
}
