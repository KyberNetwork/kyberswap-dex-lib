package efficientprop

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
	// desks is the maker gate: the desk contracts whose listings are listed
	desks *approvedDesks
}

var _ = poollist.RegisterFactoryCE1(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolsListUpdater, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &PoolsListUpdater{cfg: cfg, ethrpcClient: ethrpcClient, desks: newApprovedDesks(cfg)}, nil
}

// listing is one listing id the front answers for, with its door's address.
type listing struct {
	id   uint32
	view DoorView
	door common.Address
	// factory is the Factory of the listing's pool, zero where the read failed
	factory common.Address
}

// GetNewPools lists one pool per listing door of every configured front. A front has no listing count: listing ids
// run from 1 in order, and every id taken keeps its pool in doorView for good (a retired or delisted listing only
// loses its live flag), so the lister reads front.doorView(id) for id = 1, 2, ... until a listing the front never
// reserved (a zero pool), and factory.doorOf(id) beside it. It keeps the live listings (doorView's flags bit 0): a
// retired one never fills again, and a reserved one is listed once it goes live. A door exists only once someone
// opened it, so it keeps the doors whose immutables name that front, that listing and its pair (a door never opened
// has no code and answers none). Every read of one front is pinned to one block. Doors open at any time, so the full
// list is read on every poll, at most Config.MaxListings ids a front (logged when reached): the caller skips pools it
// already has. The scan, not the factory's ListingDoorOpened log, decides: a door is listed only while its listing is
// live, which the front's state says and no creation log does. A front whose factory cannot be read is skipped and
// logged; the poll fails only when every front does.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var pools []entity.Pool
	var errs []error
	for _, f := range u.cfg.Fronts {
		ps, err := u.listFront(ctx, f, time.Now().Unix())
		if err != nil {
			logger.WithFields(logger.Fields{"dexId": u.cfg.dexID(), "front": f.Address}).
				Warnf("efficient-prop: listing the front's doors failed: %v", err)
			errs = append(errs, err)
			continue
		}
		pools = append(pools, ps...)
	}
	if len(errs) == len(u.cfg.Fronts) {
		return nil, metadataBytes, errors.Join(errs...)
	}
	return pools, metadataBytes, nil
}

func (u *PoolsListUpdater) listFront(ctx context.Context, f Front, now int64) ([]entity.Pool, error) {
	front := common.HexToAddress(f.Address)

	// the factory must serve this front, a door forwarding to its factory's front whatever the config says, and
	// answer its door code hash (the config names none; the doors listed are the ones the factory's own doorOf names)
	var factoryFront common.Address
	var codehash, cell common.Hash
	req := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: factoryABI, Target: f.DoorFactory, Method: "FRONT"}, []any{&factoryFront}).
		AddCall(&ethrpc.Call{ABI: factoryABI, Target: f.DoorFactory, Method: "DOOR_CODEHASH"}, []any{&codehash}).
		AddCall(&ethrpc.Call{ABI: frontABI, Target: f.Address, Method: "CELL_CODEHASH"}, []any{&cell})
	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, err
	}
	if resp.BlockNumber == nil {
		return nil, ErrNoBlockNumber
	}
	if factoryFront != front {
		return nil, fmt.Errorf("%w: factory %s answers front %s", ErrFrontMismatch, f.DoorFactory,
			hexutil.Encode(factoryFront[:]))
	}
	if len(resp.Result) < 2 || !resp.Result[1] || codehash == (common.Hash{}) {
		return nil, fmt.Errorf("%w: factory %s answers no door codehash", ErrDoorCodehash, f.DoorFactory)
	}
	// the front must answer its desk code hash, which names each listing's desk contract (the maker gate), and on a
	// builder chain the one the config names, or the tracker could quote none of its listings
	if len(resp.Result) != 3 || !resp.Result[2] || cell == (common.Hash{}) ||
		u.cfg.builder() && cell != common.HexToHash(f.CellCodehash) {
		return nil, fmt.Errorf("%w: front %s answers %s, the config names %s", ErrCellCodehash, f.Address,
			hexutil.Encode(cell[:]), f.CellCodehash)
	}
	block := resp.BlockNumber

	listings, err := u.scanListings(ctx, f, codehash, cell, block)
	if err != nil {
		return nil, err
	}
	listings, err = u.matchingDoors(ctx, front, listings, block)
	if err != nil {
		return nil, err
	}

	mode, _ := f.mode() // checked by validate
	pools := make([]entity.Pool, 0, len(listings))
	for _, l := range listings {
		staticExtra, err := json.Marshal(StaticExtra{
			Front:   hexutil.Encode(front[:]),
			Listing: l.id,
			Custody: hexutil.Encode(l.view.Pool[:]),
			Door:    hexutil.Encode(l.door[:]),
			Factory: factoryOf(l.factory),
			Mode:    mode,
		})
		if err != nil {
			return nil, err
		}
		pools = append(pools, entity.Pool{
			Address:   hexutil.Encode(l.door[:]),
			Exchange:  u.cfg.dexID(),
			Type:      DexType,
			Timestamp: now,
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(l.view.Token0[:]), Swappable: true},
				{Address: hexutil.Encode(l.view.Token1[:]), Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtra),
		})
	}
	return pools, nil
}

// scanListings reads doorView and doorOf for ids from 1 in batches, until the first listing never reserved, and keeps
// the live ones of an approved desk (the maker gate, approvedDesks) whose tip cap is at the configured floor or above
// it.
func (u *PoolsListUpdater) scanListings(ctx context.Context, f Front, doorHash, cellHash common.Hash,
	block *big.Int) ([]listing, error) {
	front := common.HexToAddress(f.Address)
	// retired, delisted or not yet live, a tip cap under the configured floor, or a desk not approved (none, on a
	// listing bound to no desk): the scan goes on past it
	return u.scan(ctx, f, doorHash, block, func(v DoorView) bool {
		return v.Flags&flagLive != 0 && u.cfg.tipCapOK(tipCapWei(v.TipCap)) && v.Desk != 0 &&
			u.desks.approved(ctx, deskContractOf(front, cellHash, v.Desk))
	})
}

// scan reads doorView and doorOf for ids from 1 in batches, until the first listing never reserved, and keeps the
// listings keep answers true for.
func (u *PoolsListUpdater) scan(ctx context.Context, f Front, doorHash common.Hash, block *big.Int,
	keep func(DoorView) bool) ([]listing, error) {
	var out []listing
	limit := u.cfg.maxListings()
	for start := 1; start <= limit; start += scanBatch {
		n := min(scanBatch, limit-start+1)
		views := make([]doorViewOut, n)
		doors := make([]common.Address, n)
		req := u.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(block)
		for k := range n {
			id := big.NewInt(int64(start + k))
			req.AddCall(&ethrpc.Call{ABI: frontABI, Target: f.Address, Method: "doorView", Params: []any{id}},
				[]any{&views[k]})
			req.AddCall(&ethrpc.Call{ABI: factoryABI, Target: f.DoorFactory, Method: "doorOf", Params: []any{id}},
				[]any{&doors[k]})
		}
		resp, err := req.TryAggregate()
		if err != nil {
			return nil, err
		}
		for k := range n {
			v := views[k].View
			// doorView never reverts and answers zeros for a listing never reserved: the end of the ids
			if !resp.Result[2*k] || v.Pool == (common.Address{}) {
				return out, nil
			}
			if !resp.Result[2*k+1] || doors[k] == (common.Address{}) {
				return nil, fmt.Errorf("efficient-prop: doorOf(%d) failed", start+k)
			}
			if !keep(v) {
				continue
			}
			// the factory's door is the CREATE2 child of the listing id under its door code hash, which is what the
			// parser names a door by: a listing whose doorOf says another is left out
			if want := doorOf(f, doorHash, uint32(start+k)); hexutil.Encode(doors[k][:]) != want {
				logger.WithFields(logger.Fields{"dexId": u.cfg.dexID(), "front": f.Address, "listing": start + k}).
					Warnf("efficient-prop: the factory's doorOf is not the CREATE2 door of the listing: %s, not %s",
						hexutil.Encode(doors[k][:]), want)
				continue
			}
			out = append(out, listing{id: uint32(start + k), view: v, door: doors[k]})
		}
	}
	logger.WithFields(logger.Fields{"dexId": u.cfg.dexID(), "front": f.Address, "maxListings": limit}).
		Warnf("efficient-prop: the scan reached maxListings before the front's last listing; later ids are not read")
	return out, nil
}

// matchingDoors keeps the doors whose immutables are the front's, the listing's and the listing's pair. A door not
// yet opened has no code: its calls succeed with no data, which leaves every immutable zero, so it is not kept.
func (u *PoolsListUpdater) matchingDoors(ctx context.Context, front common.Address, listings []listing,
	block *big.Int) ([]listing, error) {
	if len(listings) == 0 {
		return nil, nil
	}
	type immutables struct {
		front, token0, token1, factory common.Address
		listing                        *big.Int
	}
	out := listings[:0:0]
	// scanBatch doors a multicall, as the scan reads its ids
	for start := 0; start < len(listings); start += scanBatch {
		batch := listings[start:min(start+scanBatch, len(listings))]
		got := make([]immutables, len(batch))
		req := u.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(block)
		for k, l := range batch {
			door := l.door.Hex()
			req.AddCall(&ethrpc.Call{ABI: doorABI, Target: door, Method: "FRONT"}, []any{&got[k].front})
			req.AddCall(&ethrpc.Call{ABI: doorABI, Target: door, Method: "LISTING"}, []any{&got[k].listing})
			req.AddCall(&ethrpc.Call{ABI: doorABI, Target: door, Method: "TOKEN0"}, []any{&got[k].token0})
			req.AddCall(&ethrpc.Call{ABI: doorABI, Target: door, Method: "TOKEN1"}, []any{&got[k].token1})
			// the pool's Factory, which holds its fee rates (takesFee); a failed read leaves it zero
			req.AddCall(&ethrpc.Call{ABI: poolFeeABI, Target: l.view.Pool.Hex(), Method: "factory"},
				[]any{&got[k].factory})
		}
		resp, err := req.TryAggregate()
		if err != nil {
			return nil, err
		}
		for k, l := range batch {
			g := got[k]
			ok := resp.Result[5*k] && resp.Result[5*k+1] && resp.Result[5*k+2] && resp.Result[5*k+3] &&
				g.front == front && g.listing != nil && g.listing.Cmp(big.NewInt(int64(l.id))) == 0 &&
				g.token0 == l.view.Token0 && g.token1 == l.view.Token1 &&
				g.token0 != (common.Address{}) && g.token0.Cmp(g.token1) < 0
			if ok {
				if resp.Result[5*k+4] {
					l.factory = g.factory
				}
				out = append(out, l)
			}
		}
	}
	return out, nil
}

// factoryOf is a pool's Factory as the static extra holds it: empty where it was not read.
func factoryOf(a common.Address) string {
	if a == (common.Address{}) {
		return ""
	}
	return hexutil.Encode(a[:])
}
