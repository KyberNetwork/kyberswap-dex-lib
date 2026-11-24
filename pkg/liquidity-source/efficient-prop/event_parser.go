package efficientprop

import (
	"context"
	"errors"
	"math/big"
	"slices"
	"sync"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

// EventParser maps the logs that move a listing's price or fill to its door, the pool, so the pool is refreshed at
// once rather than at its next scheduled refresh (PoolTracker.NextUpdateAt):
//   - the front's logs naming a listing: its fills (PoolFill, through any door or entry), its book and controls posts
//     (DepthPosted, ControlsPosted), its status, settings, key, binding and asset module (PoolStatus, SettingsSet,
//     MakerKeySet, ListingBound, SourceDarkened, LeaseSet, AssetModuleSet), and its birth and end (PoolReserved,
//     PoolRegistered, PoolRetired, ListingDelisted, ListingRelisted). The door is the factory's CREATE2 address of the
//     listing id, computed here from the configured factory and the door codehash it answers (DOOR_CODEHASH(), read
//     once).
//   - the front's logs naming a desk, its stop, revive and key changes (DeskStopped, DeskRevived, DeskKeySet): the
//     doors of every listing bound to that desk.
//   - the front's pause and unpause of every fill (AllPaused, AllUnpaused), a pair's retirement (PairRetired), a
//     pair's or the protocol's asset module (AssetModuleSet of listing 0) and a lease set for every listing (LeaseSet
//     of listing 0): every door of the front. The parser does not learn which pair a listing trades on, so a pair's
//     event refreshes the front's every door.
//   - a desk's quote contract's DeskPush (a post of the desk's words) and LaneFill: the doors of every listing whose
//     updates that contract holds, however many share it. The contract is no configured address: the parser learns
//     each listing's from the front by its front's slot rule (doorView's desk, then deskCellOf), and
//     learns again when a contract or desk it does not know logs, or after a binding, a source change, a desk's
//     opening or a listing's reservation. The logs name no listing, so a log refreshes every listing of the contract.
//   - a stored price lane's Posted (a post of its prints), where the lane is a source its front's config approves: the
//     doors of every listing that reads a print (PRINT or SESSION, not tied) bound to a source on that lane, its
//     primary or its backup. A lane is a configured address; the parser learns each listing's binding from the front's
//     storage (record word 10 and the entries it names), and learns again after a binding, as for a desk, or a source
//     change (a SettingsSet of kind 34, which setSource logs). A post that stored nothing maps to nothing.
//   - a fill's Swapped (IPropAMM's event, once per settled fill). Its emitter follows the front's build. On a storage
//     build the door called logs it: that door. On a builder build (Ethereum) the filling listing's quote contract
//     logs it and no door does, naming the front's caller its sender: where the sender is a door of a listing that
//     contract holds, that door, and else (a fill through the front's own entries) every door of the contract, as
//     for LaneFill. The topic is not ours alone (any IPropAMM venue logs it), so a Swapped of an address the parser
//     does not know maps to nothing and prompts no learning: a new desk's contract is learned at its binding.
//
// The parser learns every listing the front reserved, live or not, so a listing relisted later is mapped as it is.
//
// On a builder chain (Ethereum) a desk's update lands only with the block that fills against it and comes in /state
// (Front.Titan), not by a log: no desk logs DeskPush there (the storage build's) and no stored price lane posts. So
// the parser maps neither, and reads no listing's price binding; every other log above maps as it does elsewhere.
type EventParser struct {
	cfg          *Config
	cfgErr       error
	ethrpcClient *ethrpc.Client
	fronts       map[common.Address]Front
	// builder: the chain is a builder chain, whose desks and lanes post by no log
	builder bool
	// hashes are the code hashes the fronts and their door factories answer, read once
	hashes *codeHashes

	mu sync.Mutex
	// known is what the parser last learned of the fronts (learnedAt); stale after a binding, a source change, a desk's
	// opening or a listing's reservation
	known     learned
	learnedAt time.Time
	stale     bool
}

// learned is what the parser learned of the fronts at one block: the doors of the listings whose updates each desk
// quote contract holds (accounts: by the slot rule, one or several a contract), of each desk's, of each approved
// lane's (every lane a front approves, bound to a listing or not), and of every listing of each front.
type learned struct {
	accounts map[common.Address][]string
	desks    map[deskKey][]string
	lanes    map[common.Address][]string
	all      map[common.Address][]string
	// lanesUnread: a front's listings' binding could not be read, so its lanes are not mapped, and any log makes the
	// parser learn again once the learning is relearnAfter old
	lanesUnread bool
}

// deskKey is one front's desk.
type deskKey struct {
	front common.Address
	desk  uint32
}

var _ = poolfactory.RegisterFactoryCE(DexType, NewPoolFactory)

// relearnAfter is the least time between two learnings of the fronts prompted by a cell or desk the parser does not
// know.
const relearnAfter = 10 * time.Second

var (
	topicDeskPush = cellABI.Events["DeskPush"].ID
	topicLaneFill = cellABI.Events["LaneFill"].ID
	topicSwapped  = cellABI.Events["Swapped"].ID // the door's on a storage build: one event, IPropAMM's
	topicPosted   = laneABI.Events["Posted"].ID

	topicSettingsSet = frontABI.Events["SettingsSet"].ID

	// listingTopic is the topic index of the listing id in each front event that names one
	listingTopic = func() map[common.Hash]int {
		m := map[common.Hash]int{}
		for _, e := range []string{"PoolFill", "PoolReserved", "PoolRegistered", "AssetModuleSet"} {
			m[frontABI.Events[e].ID] = 2
		}
		for _, e := range []string{"SettingsSet", "PoolRetired", "ListingDelisted", "ListingRelisted", "ListingBound",
			"PoolStatus", "DepthPosted", "ControlsPosted", "SourceDarkened", "LeaseSet", "MakerKeySet"} {
			m[frontABI.Events[e].ID] = 1
		}
		return m
	}()
	// deskTopics are the front events naming a desk (topic 1) that change what its listings fill
	deskTopics = map[common.Hash]bool{frontABI.Events["DeskStopped"].ID: true,
		frontABI.Events["DeskRevived"].ID: true, frontABI.Events["DeskKeySet"].ID: true}
	// frontTopics are the front events that change what every listing of the front fills, or every listing of a pair,
	// which the parser does not map to its listings: the pause and unpause of every fill, and a pair's retirement
	frontTopics = map[common.Hash]bool{frontABI.Events["AllPaused"].ID: true, frontABI.Events["AllUnpaused"].ID: true,
		frontABI.Events["PairRetired"].ID: true}
	// everyAtZero are the front events naming a listing that, with listing 0, change what every listing of the front
	// fills, or every listing of a pair: every lease cut or that cut lifted (LeaseSet), a pair's asset module bound or
	// removed for good, or the protocol's (AssetModuleSet)
	everyAtZero = map[common.Hash]bool{frontABI.Events["LeaseSet"].ID: true,
		frontABI.Events["AssetModuleSet"].ID: true}
	// rebinds are the front events after which a desk's listings, or the front's, may be others
	rebinds = map[common.Hash]bool{frontABI.Events["ListingBound"].ID: true, frontABI.Events["DeskOpened"].ID: true,
		frontABI.Events["PoolReserved"].ID: true}
)

func NewPoolFactory(cfg *Config, ethrpcClient *ethrpc.Client) *EventParser {
	p := &EventParser{cfg: cfg, ethrpcClient: ethrpcClient, fronts: map[common.Address]Front{},
		builder: builderChains[cfg.ChainID], hashes: newCodeHashes(ethrpcClient)}
	if p.cfgErr = cfg.validate(); p.cfgErr == nil {
		for _, f := range cfg.Fronts {
			p.fronts[common.HexToAddress(f.Address)] = f
		}
	}
	return p
}

// Decode maps each log to the doors whose state it moves.
func (p *EventParser) Decode(ctx context.Context, logs []types.Log) (map[string][]types.Log, error) {
	if p.cfgErr != nil {
		return nil, p.cfgErr
	}
	addressLogs := make(map[string][]types.Log)
	for _, log := range logs {
		doors, err := p.doorsOf(ctx, log)
		if err != nil {
			return nil, err
		}
		for _, door := range doors {
			addressLogs[door] = append(addressLogs[door], log)
		}
	}
	return addressLogs, nil
}

func (p *EventParser) doorsOf(ctx context.Context, log types.Log) ([]string, error) {
	if len(log.Topics) == 0 {
		return nil, nil
	}
	topic := log.Topics[0]
	if f, ok := p.fronts[log.Address]; ok {
		if rebinds[topic] || sourceChange(log) {
			p.mu.Lock()
			p.stale = true
			p.mu.Unlock()
		}
		every := func() []string {
			return p.learnedDoors(ctx, func(l learned) ([]string, bool) {
				doors, ok := l.all[log.Address]
				return doors, ok
			})
		}
		if frontTopics[topic] {
			return every(), nil
		}
		if deskTopics[topic] {
			if len(log.Topics) < 2 || log.Topics[1].Big().BitLen() > 32 {
				return nil, nil
			}
			k := deskKey{front: log.Address, desk: uint32(log.Topics[1].Big().Uint64())}
			return p.learnedDoors(ctx, func(l learned) ([]string, bool) {
				doors, ok := l.desks[k]
				return doors, ok
			}), nil
		}
		k, ok := listingTopic[topic]
		if !ok || len(log.Topics) <= k {
			return nil, nil
		}
		id := log.Topics[k].Big()
		if id.Sign() == 0 && everyAtZero[topic] {
			return every(), nil
		}
		if id.Sign() == 0 || id.BitLen() > 24 {
			return nil, nil // no listing
		}
		hash, err := p.hashes.door(ctx, f)
		if err != nil {
			// no door can be named for this log; the other logs of the batch are mapped as usual
			p.skipFront(f, err)
			return nil, nil
		}
		return []string{doorOf(f, hash, uint32(id.Uint64()))}, nil
	}
	switch topic {
	case topicSwapped:
		if !p.builder {
			// a storage build: the door called logs its own fill
			return []string{hexutil.Encode(log.Address[:])}, nil
		}
		// a builder build: the filling listing's quote contract logs the fill, and names the front's caller its
		// sender (topic 1), behind a door that door
		doors := p.knownDoors(ctx, func(l learned) []string { return l.accounts[log.Address] })
		if len(log.Topics) > 1 {
			if sender := hexutil.Encode(log.Topics[1][common.HashLength-common.AddressLength:]); slices.Contains(doors,
				sender) {
				return []string{sender}, nil
			}
		}
		return doors, nil
	case topicDeskPush, topicLaneFill:
		if p.builder && topic == topicDeskPush {
			return nil, nil
		}
		return p.learnedDoors(ctx, func(l learned) ([]string, bool) {
			doors, ok := l.accounts[log.Address]
			return doors, ok
		}), nil
	case topicPosted:
		// a post that stored no feed moved no print
		if p.builder || !p.approvedLane(log.Address) || new(big.Int).SetBytes(log.Data).Sign() == 0 {
			return nil, nil
		}
		return p.learnedDoors(ctx, func(l learned) ([]string, bool) {
			doors, ok := l.lanes[log.Address]
			return doors, ok
		}), nil
	}
	return nil, nil
}

// sourceChange is whether a front's log is a listing's source change: setSource logs SettingsSet(listing, kind,
// value) with kind 34 (its first data word), and no ListingBound.
func sourceChange(log types.Log) bool {
	return log.Topics[0] == topicSettingsSet && len(log.Data) >= common.HashLength &&
		new(big.Int).SetBytes(log.Data[:common.HashLength]).Cmp(big.NewInt(kindSource)) == 0
}

// approvedLane is whether lane is a price source some front's config approves.
func (p *EventParser) approvedLane(lane common.Address) bool {
	for _, f := range p.fronts {
		if f.approves(lane) {
			return true
		}
	}
	return false
}

// learnedDoors is the doors find picks from what the parser learned of the fronts, learning again first where that is
// stale, never learned, or where find knows nothing and the last learning is relearnAfter old. A failed learning is
// logged and the doors last learned are kept.
func (p *EventParser) learnedDoors(ctx context.Context, find func(learned) ([]string, bool)) []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	doors, known := find(p.known)
	if p.stale || p.known.all == nil || (!known || p.known.lanesUnread) && time.Since(p.learnedAt) >= relearnAfter {
		l, err := p.learn(ctx)
		if err != nil {
			logger.WithFields(logger.Fields{"dexId": p.cfg.dexID()}).
				Warnf("efficient-prop: learning the fronts' listings failed: %v", err)
			return doors
		}
		p.known, p.learnedAt, p.stale = l, time.Now(), false
		doors, _ = find(p.known)
	}
	return doors
}

// knownDoors is the doors find picks from what the parser learned of the fronts, learning first only where that is
// stale or never learned: a log whose topic other venues share (Swapped) from an address the parser does not know
// prompts no learning of its own. A failed learning is logged and the doors last learned are kept.
func (p *EventParser) knownDoors(ctx context.Context, find func(learned) []string) []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stale || p.known.all == nil {
		l, err := p.learn(ctx)
		if err != nil {
			logger.WithFields(logger.Fields{"dexId": p.cfg.dexID()}).
				Warnf("efficient-prop: learning the fronts' listings failed: %v", err)
			return find(p.known)
		}
		p.known, p.learnedAt, p.stale = l, time.Now(), false
	}
	return find(p.known)
}

// learn reads every listing each front reserved, live or not (the lister's scan, at one block), the quote contract
// of each listing bound to a desk (its front's slot rule, each distinct call once) and, where the front approves a
// source, each print-reading listing's binding.
func (p *EventParser) learn(ctx context.Context) (learned, error) {
	u := &PoolsListUpdater{cfg: p.cfg, ethrpcClient: p.ethrpcClient}
	l := learned{accounts: map[common.Address][]string{}, desks: map[deskKey][]string{},
		lanes: map[common.Address][]string{}, all: map[common.Address][]string{}}
	head, err := p.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		return l, err
	}
	block := new(big.Int).SetUint64(head)
	for _, f := range p.cfg.Fronts {
		front := common.HexToAddress(f.Address)
		// a front whose door hash cannot be read, or on a builder chain whose desk hash is not the config's, is left
		// out of this learning; the others carry on
		doorHash, err := p.hashes.door(ctx, f)
		if err != nil {
			p.skipFront(f, err)
			continue
		}
		listings, err := u.scan(ctx, f, doorHash, block, func(DoorView) bool { return true })
		if err != nil {
			p.skipFront(f, err)
			continue
		}
		var rule deskRule
		if p.builder {
			if rule, err = slotRuleOf(ctx, p.hashes, &f); err != nil {
				p.skipFront(f, err)
				continue
			}
		}
		// the slot rule's calls, each sent once (several listings, as a desk's under one quote contract per desk,
		// may share one), and each bound listing's door and call
		type bound struct {
			b    binding
			door string
			read int
		}
		var reads []call
		var holders []any
		var bounds []bound
		sent := map[string]int{}
		all := []string{}
		for _, li := range listings {
			door := doorOf(f, doorHash, li.id)
			all = append(all, door)
			if li.view.Desk == 0 {
				continue
			}
			k := deskKey{front: front, desk: li.view.Desk}
			l.desks[k] = append(l.desks[k], door)
			b := binding{listing: li.id, desk: li.view.Desk, wordIndex: li.view.WordIndex}
			c, holder := rule.read(front, b)
			data, err := c.abi.Pack(c.method, c.params...)
			if err != nil {
				return l, err
			}
			key := string(c.target[:]) + string(data)
			r, ok := sent[key]
			if !ok {
				r = len(reads)
				sent[key] = r
				reads, holders = append(reads, c), append(holders, holder)
			}
			bounds = append(bounds, bound{b: b, door: door, read: r})
		}
		l.all[front] = all
		lanes, err := p.learnLanes(ctx, f, doorHash, block, listings)
		if err != nil {
			// the front's lanes stay unmapped until a later learning, which any log asks for once this one is
			// relearnAfter old; the desks and cells this learning reads are kept
			logger.WithFields(logger.Fields{"dexId": p.cfg.dexID(), "front": f.Address}).
				Warnf("efficient-prop: reading the listings' price sources failed: %v", err)
			l.lanesUnread = true
		}
		for lane, doors := range lanes {
			l.lanes[lane] = append(l.lanes[lane], doors...)
		}
		if len(reads) == 0 {
			continue
		}
		req := p.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(block)
		for k, c := range reads {
			req.AddCall(&ethrpc.Call{ABI: *c.abi, Target: hexutil.Encode(c.target[:]), Method: c.method,
				Params: c.params}, []any{holders[k]})
		}
		resp, err := req.TryAggregate()
		if err != nil {
			return l, err
		}
		for _, b := range bounds {
			if !resp.Result[b.read] {
				continue
			}
			cell, _ := holders[b.read].(*common.Address)
			if cell == nil {
				continue
			}
			if p.builder {
				// the per-desk rule: only the desk's CREATE2 child
				s, ok := rule.slots(front, b.b, holders[b.read])
				if !ok {
					continue
				}
				cell = &s.account
			} else if *cell == (common.Address{}) {
				continue
			}
			l.accounts[*cell] = append(l.accounts[*cell], b.door)
		}
	}
	return l, nil
}

// skipFront logs a front the parser leaves out of a learning.
func (p *EventParser) skipFront(f Front, err error) {
	logger.WithFields(logger.Fields{"dexId": p.cfg.dexID(), "front": f.Address}).
		Warnf("efficient-prop: the front's listings are not learned: %v", err)
}

// laneReads is how many storage reads learnLanes sends in one JSON-RPC batch: the lister's batch, under what a
// provider that caps a batch at 100 requests takes.
const laneReads = scanBatch

// learnLanes is, for each source front f approves, the doors of its listings that read a print (PRINT or SESSION, not
// tied) bound to that source, its primary or its backup (record word 10, each id's entry naming the source's contract,
// read from the front's storage at block, in batches of laneReads), and the source with no doors where none is bound
// to it. A word the node does not answer leaves its listing unmapped; a batch the node or its provider refuses fails
// the read. On a builder chain, where no lane posts by a log, nothing.
func (p *EventParser) learnLanes(ctx context.Context, f Front, doorHash common.Hash, block *big.Int,
	listings []listing) (map[common.Address][]string, error) {
	if len(f.ApprovedSources) == 0 || p.builder {
		return nil, nil
	}
	lanes := map[common.Address][]string{}
	for _, a := range f.ApprovedSources {
		if lane := common.HexToAddress(a); lanes[lane] == nil {
			lanes[lane] = []string{}
		}
	}
	front := common.HexToAddress(f.Address)
	read := func(slots []common.Hash) ([]*big.Int, error) {
		out := make([]*big.Int, len(slots))
		for lo := 0; lo < len(slots); lo += laneReads {
			hi := min(lo+laneReads, len(slots))
			words := make([]hexutil.Bytes, hi-lo)
			batch := make([]rpc.BatchElem, hi-lo)
			for k := range batch {
				batch[k] = rpc.BatchElem{Method: "eth_getStorageAt",
					Args: []any{front, slots[lo+k], hexutil.EncodeBig(block)}, Result: &words[k]}
			}
			if err := eth.BatchCallWithRetry(ctx, p.ethrpcClient.GetETHClient().Client(), batch,
				eth.DefaultBatchRetry); err != nil {
				return nil, err
			}
			for k, b := range batch {
				if b.Error == nil && len(words[k]) == common.HashLength {
					out[lo+k] = new(big.Int).SetBytes(words[k])
				}
			}
		}
		return out, nil
	}
	var reading []listing
	var slots []common.Hash
	for _, li := range listings {
		if printed(li.view.Scopes) && (li.view.Scopes>>1)&15 != tiedScopes {
			reading = append(reading, li)
			slots = append(slots, sourceIdsSlot(li.id))
		}
	}
	if len(reading) == 0 {
		return lanes, nil
	}
	ids, err := read(slots)
	if err != nil {
		return nil, err
	}
	var sources []uint32
	for _, w := range ids {
		for _, id := range []uint32{uint32(bits(w, 0, 32)), uint32(bits(w, 32, 32))} {
			if id != 0 && !slices.Contains(sources, id) {
				sources = append(sources, id)
			}
		}
	}
	entrySlots := make([]common.Hash, len(sources))
	for k, id := range sources {
		entrySlots[k] = sourceSlot(id)
	}
	entries, err := read(entrySlots)
	if err != nil {
		return nil, err
	}
	for k, li := range reading {
		door := doorOf(f, doorHash, li.id)
		for _, id := range []uint32{uint32(bits(ids[k], 0, 32)), uint32(bits(ids[k], 32, 32))} {
			if id == 0 {
				continue
			}
			e := entries[slices.Index(sources, id)]
			if e == nil {
				continue
			}
			if lane := common.BigToAddress(e); f.approves(lane) && !slices.Contains(lanes[lane], door) {
				lanes[lane] = append(lanes[lane], door)
			}
		}
	}
	return lanes, nil
}

// doorOf is a listing's door: the CREATE2 address of the listing id under the front's door factory and hash, the
// doors' creation code hash the factory answers (DOOR_CODEHASH()), as the factory's own doorOf computes it.
func doorOf(f Front, hash common.Hash, listing uint32) string {
	a := crypto.CreateAddress2(common.HexToAddress(f.DoorFactory), common.BigToHash(big.NewInt(int64(listing))),
		hash.Bytes())
	return hexutil.Encode(a[:])
}

// DecodePoolCreated is not used: the lister finds doors by the front's state (PoolsListUpdater.GetNewPools).
func (p *EventParser) DecodePoolCreated(types.Log) (*entity.Pool, error) {
	return nil, errors.New("efficient-prop: doors are listed by the front's state, not by a creation log")
}

func (p *EventParser) IsEventSupported(common.Hash) bool { return false }
