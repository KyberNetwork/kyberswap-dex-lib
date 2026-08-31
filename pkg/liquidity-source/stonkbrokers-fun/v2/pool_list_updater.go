package stonkbrokersfunv2

import (
	"context"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, client *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: client}
}

// launchRaw mirrors StonkSafeLaunchpadV2.Launch's field names/types exactly
// -- go-ethereum's abi package unpacks a tuple return into a struct by
// matching PascalCase field names to the ABI's tuple component names.
// Verified against a live getLaunch(176) call.
type launchRaw struct {
	Token              common.Address
	Creator            common.Address
	StartMcapUsd8      uint64
	GradMcapUsd8       uint64
	StartTaxBps        uint16
	DecayPerMinuteBps  uint16
	CreatorFeeBpsSnap  uint16
	ProtocolFeeBpsSnap uint16
	WindowSecs         uint32
	StartTime          uint64
	Deadline           uint64
	ExternalToken      bool
	SellsEnabled       bool
	Armed              bool
	Graduated          bool
	Bonded             bool
	Aborted            bool
	LoadedSupply       *big.Int
	VQuote             *big.Int
	VToken             *big.Int
	RealQuote          *big.Int
	BuyCount           *big.Int
}

// modesRaw mirrors StonkSafeLaunchpadV2.LaunchModes.
type modesRaw struct {
	UnsoldMode uint8
	EoaOnly    bool
	OpenEnded  bool
	PostTaxBps uint16
	BondVenue  uint8
	MaxBuyPpm  uint32
	IcoBoost   bool
}

// getLaunchResult / modesOfResult wrap the single-tuple returns of
// getLaunch(uint256) and modesOf(uint256). go-ethereum's Arguments.isTuple()
// is len(outputs) > 1, so a method returning exactly one `tuple` takes the
// copyAtomic path, which assigns into *field 0* of a struct destination rather
// than the struct itself. Unpacking straight into launchRaw would therefore try
// to write the whole tuple into launchRaw.Token ([20]byte) and panic in
// abi.setArray. The extra level of nesting puts the tuple at field 0.
type getLaunchResult struct {
	Launch launchRaw
}

type modesOfResult struct {
	Modes modesRaw
}

// padStatic is the pad-wide (immutable, same for every launch on this pad)
// info fetched once per updater pass.
type padStatic struct {
	quote          common.Address
	quoteDecimals  uint8
	isWethLane     bool
	quoteIsToken0  bool
	quoteUsdFeed   common.Address
	twapPool       common.Address
	twapWindowSecs uint32
	ethUsdFeed     common.Address
	bufferTaxBps   *big.Int
	launchCount    *big.Int
}

// PoolAddress is the synthetic key for one launch. Launch ids are per-pad, so
// neither the pad address nor the id identifies a pool on its own.
//
// The separator is "_" and deliberately NOT "#": entity.Pool.Address travels
// through URL query strings (router-service's poolIds parameter, among others),
// and "#" starts a fragment there, so a caller passing the key verbatim silently
// loses the id and matches nothing. "_" is unreserved in RFC 3986. Same shape as
// fluid/dex-v2's encodeFluidDexV2PoolAddress.
func PoolAddress(pad string, launchID uint64) string {
	return pad + "_" + strconv.FormatUint(launchID, 10)
}

// GetNewPools implements the on-chain, cursor-based discovery: for each of the
// 8 fixed Smart Launch V2 pads, page forward from the last known
// launchId to the pad's current launchCount(), emitting one entity.Pool per
// newly discovered (pad, launchId).
//
// Per AGENTS.md: reserves are NOT set here (left "0","0"; the tracker fills
// them on the next refresh) and token decimals are NOT fetched here (the
// token-metadata pipeline populates them after listing) --
// QuoteDecimals in StaticExtra comes from the PAD's own quoteDecimals()
// view, not an ERC20 introspection call, so it is available immediately for
// the math package without violating that rule.
func (l *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	cursor := map[string]uint64{}
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &cursor); err != nil {
			return nil, metadataBytes, err
		}
	}

	var pools []entity.Pool
	now := time.Now().Unix()

	for _, rawPad := range l.config.Pads {
		pad := strings.ToLower(strings.TrimSpace(rawPad))
		next := cursor[pad]
		if next == 0 {
			next = 1
		}

		ps, err := l.fetchPadStatic(ctx, pad)
		if err != nil {
			logger.WithFields(logger.Fields{"err": err, "pad": pad, "dex": DexType}).
				Errorf("stonkbrokers-fun-v2: fetch pad static info")
			return pools, metadataBytes, err
		}

		count := ps.launchCount.Uint64()
		if count < next {
			continue
		}

		newPools, err := l.fetchLaunches(ctx, pad, ps, next, count, now)
		if err != nil {
			logger.WithFields(logger.Fields{"err": err, "pad": pad, "dex": DexType}).
				Errorf("stonkbrokers-fun-v2: fetch launches")
			return pools, metadataBytes, err
		}
		pools = append(pools, newPools...)
		cursor[pad] = count + 1
	}

	newMetadata, err := json.Marshal(cursor)
	if err != nil {
		return pools, metadataBytes, err
	}
	return pools, newMetadata, nil
}

func (l *PoolsListUpdater) fetchPadStatic(ctx context.Context, pad string) (*padStatic, error) {
	var ps padStatic
	req := l.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{ABI: PadABI, Target: pad, Method: methodLaunchCount}, []any{&ps.launchCount})
	req.AddCall(&ethrpc.Call{ABI: PadABI, Target: pad, Method: methodQuote}, []any{&ps.quote})
	req.AddCall(&ethrpc.Call{ABI: PadABI, Target: pad, Method: methodQuoteDecimals}, []any{&ps.quoteDecimals})
	req.AddCall(&ethrpc.Call{ABI: PadABI, Target: pad, Method: methodIsWethLane}, []any{&ps.isWethLane})
	req.AddCall(&ethrpc.Call{ABI: PadABI, Target: pad, Method: methodQuoteIsToken0}, []any{&ps.quoteIsToken0})
	req.AddCall(&ethrpc.Call{ABI: PadABI, Target: pad, Method: methodQuoteUsdFeed}, []any{&ps.quoteUsdFeed})
	req.AddCall(&ethrpc.Call{ABI: PadABI, Target: pad, Method: methodTwapPool}, []any{&ps.twapPool})
	req.AddCall(&ethrpc.Call{ABI: PadABI, Target: pad, Method: methodTwapWindowSecs}, []any{&ps.twapWindowSecs})
	req.AddCall(&ethrpc.Call{ABI: PadABI, Target: pad, Method: methodEthUsdFeed}, []any{&ps.ethUsdFeed})
	req.AddCall(&ethrpc.Call{ABI: PadABI, Target: pad, Method: methodBufferTaxBps}, []any{&ps.bufferTaxBps})
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	return &ps, nil
}

func (l *PoolsListUpdater) fetchLaunches(
	ctx context.Context, pad string, ps *padStatic, fromID, toID uint64, ts int64,
) ([]entity.Pool, error) {
	n := int(toID-fromID) + 1
	launches := make([]getLaunchResult, n)
	modes := make([]modesOfResult, n)
	bufferSecs := make([]uint32, n)

	req := l.ethrpcClient.NewRequest().SetContext(ctx)
	for i := 0; i < n; i++ {
		id := new(big.Int).SetUint64(fromID + uint64(i))
		req.AddCall(&ethrpc.Call{ABI: PadABI, Target: pad, Method: methodGetLaunch, Params: []any{id}}, []any{&launches[i]})
		req.AddCall(&ethrpc.Call{ABI: PadABI, Target: pad, Method: methodModesOf, Params: []any{id}}, []any{&modes[i]})
		req.AddCall(&ethrpc.Call{ABI: PadABI, Target: pad, Method: methodBufferSecsOf, Params: []any{id}}, []any{&bufferSecs[i]})
	}
	resp, err := req.TryAggregate()
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, n)
	for i := 0; i < n; i++ {
		base := i * 3
		if base+2 >= len(resp.Result) || !resp.Result[base] || !resp.Result[base+1] || !resp.Result[base+2] {
			// getLaunch/modesOf/bufferSecsOf reverted for this id -- skip it,
			// a later refresh will pick it up once/if it becomes readable.
			continue
		}
		id := fromID + uint64(i)
		lr := launches[i].Launch
		mr := modes[i].Modes

		// eoaOnly launches revert NotEoa() for any contract caller, and an
		// aggregator always reaches buy() from the executor, so they can never
		// be routed. Drop them here rather than indexing pools that only exist
		// to be refused: 60 of 362 live launches set the flag. The cursor pages
		// by launch id, not by pools emitted, so skipping does not lose ground.
		if mr.EoaOnly {
			continue
		}

		staticExtra := StaticExtra{
			Pad:               pad,
			Lens:              strings.ToLower(l.config.Lens),
			LaunchID:          strconv.FormatUint(id, 10),
			IsWethLane:        ps.isWethLane,
			QuoteDecimals:     ps.quoteDecimals,
			BufferTaxBps:      uint16(ps.bufferTaxBps.Uint64()),
			StartTaxBps:       lr.StartTaxBps,
			DecayPerMinuteBps: lr.DecayPerMinuteBps,
			BufferSecs:        bufferSecs[i],
			WindowSecs:        lr.WindowSecs,
			StartTime:         lr.StartTime,
			Deadline:          lr.Deadline,
			OpenEnded:         mr.OpenEnded,
			EoaOnly:           mr.EoaOnly,
			PostTaxBps:        mr.PostTaxBps,
			MaxBuyPpm:         mr.MaxBuyPpm,
			GradMcapUsd8:      lr.GradMcapUsd8,
			LoadedSupply:      lr.LoadedSupply.String(),
			QuoteIsToken0:     ps.quoteIsToken0,
		}
		if (ps.quoteUsdFeed != common.Address{}) {
			staticExtra.QuoteUsdFeed = strings.ToLower(ps.quoteUsdFeed.Hex())
		}
		if (ps.twapPool != common.Address{}) {
			staticExtra.TwapPool = strings.ToLower(ps.twapPool.Hex())
			staticExtra.TwapWindowSecs = ps.twapWindowSecs
		}
		if (ps.ethUsdFeed != common.Address{}) {
			staticExtra.EthUsdFeed = strings.ToLower(ps.ethUsdFeed.Hex())
		}

		seBytes, err := json.Marshal(staticExtra)
		if err != nil {
			return nil, err
		}

		pools = append(pools, entity.Pool{
			Address:   PoolAddress(pad, id),
			Exchange:  l.config.DexID,
			Type:      DexType,
			Timestamp: ts,
			Reserves:  entity.PoolReserves{"0", "0"}, // tracker fills real reserves next refresh
			Tokens: []*entity.PoolToken{
				{Address: strings.ToLower(lr.Token.Hex()), Swappable: true},
				{Address: strings.ToLower(ps.quote.Hex()), Swappable: true},
			},
			StaticExtra: string(seBytes),
		})
	}
	return pools, nil
}
