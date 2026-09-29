package deepstateob

import (
	"context"
	"math"
	"math/big"
	"sort"
	"strconv"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}, nil
}

// GetNewPoolState fetches a fresh book snapshot. StaticExtra.Lens set means
// one round trip via getBookViaLens; otherwise it falls back to the naive
// per-level BFS walk (getBookNaive), which costs O(tree depth) round trips.
func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{"poolAddress": p.Address, "dexID": t.config.DexID})

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return entity.Pool{}, err
	}

	token0 := common.HexToAddress(p.Tokens[0].Address)
	token1 := common.HexToAddress(p.Tokens[1].Address)

	var (
		feeBps               uint16
		epoch                *big.Int
		bidLeaves, askLeaves []decodedNode
		blockNumber          uint64
		err                  error
	)

	if staticExtra.Lens != "" {
		feeBps, epoch, bidLeaves, askLeaves, blockNumber, err = t.getBookViaLens(ctx, staticExtra.Lens, token0, token1)
	} else {
		feeBps, epoch, bidLeaves, askLeaves, blockNumber, err = t.getBookNaive(ctx, staticExtra.Router, token0, token1)
	}
	if err != nil {
		l.WithFields(logger.Fields{"error": err}).Error("failed to fetch deepstate-ob book")
		return entity.Pool{}, err
	}

	dec0, dec1 := p.Tokens[0].Decimals, p.Tokens[1].Decimals
	extra := Extra{
		Extra: orderbook.Extra{
			LevelsFrom: [2][]orderbook.Level{
				buildLevels(bidLeaves, dec0, dec1, false), // swap FROM token0 -> matches resting bids
				buildLevels(askLeaves, dec0, dec1, true),  // swap FROM token1 -> matches resting asks
			},
		},
		Epoch: epoch.String(),
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}

	// reserve0/1 are liquidity-depth estimates, not custodied balances --
	// DeepstateV1 has no reserve getter. Mirrors kuru-ob's approach.
	var reserve0, reserve1 float64
	for _, lvl := range extra.LevelsFrom[1] {
		reserve0 += lvl.Size() * lvl.Price()
	}
	for _, lvl := range extra.LevelsFrom[0] {
		reserve1 += lvl.Size() * lvl.Price()
	}

	p.SwapFee = float64(feeBps) / 10_000
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		strconv.FormatFloat(reserve0*math.Pow10(int(dec0)), 'f', 0, 64),
		strconv.FormatFloat(reserve1*math.Pow10(int(dec1)), 'f', 0, 64),
	}
	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber

	l.WithFields(logger.Fields{
		"epoch": epoch.String(), "bids": len(bidLeaves), "asks": len(askLeaves),
		"viaLens": staticExtra.Lens != "",
	}).Info("finished updating deepstate-ob pool state")
	return p, nil
}

// getBookViaLens fetches the entire book in one RPC round trip via a
// deployed DeepstateBookLens.getBook. Primary path once StaticExtra.Lens is
// set. Uses TryBlockAndAggregate (not plain Call) so the block number that
// produced this snapshot is available without a second call.
func (t *PoolTracker) getBookViaLens(
	ctx context.Context, lens string, token0, token1 common.Address,
) (feeBps uint16, epoch *big.Int, bidLeaves, askLeaves []decodedNode, blockNumber uint64, err error) {
	var book LensBookRPC
	// go-ethereum's abi.Arguments.Copy treats a single named output as a
	// plain value, not a tuple, so it overwrites book.Field(0) with the
	// whole return value instead of matching fields by name. Wrapping the
	// target routes it through setStruct's field-by-field copy instead --
	// same fix as pkg/liquidity-source/uniswap/v4/hooks/aegisprop/hook.go.
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI: deepstateBookLensABI, Target: lens, Method: methodGetBook,
		Params: []any{token0, token1, big.NewInt(lensMaxNodes)},
	}, []any{&struct{ *LensBookRPC }{&book}}).TryBlockAndAggregate()
	if err != nil {
		return 0, nil, nil, nil, 0, err
	}
	if !resp.Result[0] {
		return 0, nil, nil, nil, 0, ErrRPCCallReverted
	}

	if book.Bid.Truncated || book.Ask.Truncated {
		logger.WithFields(logger.Fields{
			"lens": lens, "bidTruncated": book.Bid.Truncated, "askTruncated": book.Ask.Truncated,
		}).Warn("deepstate-ob: lens BFS walk truncated at lensMaxNodes, book snapshot is partial")
	}

	return book.FeeBps, book.Epoch, sideToLeaves(book.Bid), sideToLeaves(book.Ask), resp.BlockNumber.Uint64(), nil
}

func sideToLeaves(side LensSideRPC) []decodedNode {
	leaves := make([]decodedNode, len(side.Ticks))
	for i, tick := range side.Ticks {
		q := new(uint256.Int)
		q.SetFromBig(side.Quantities[i])
		leaves[i] = decodedNode{tick: tick, quantity: q}
	}
	return leaves
}

// getBookNaive is the fallback path for a router with no lens deployed: one
// round trip for feeConfig+poolEpoch, one for roots, then one round trip per
// BFS tree level per side (see bfsWalkLeaves). Costs O(tree depth) round
// trips instead of getBookViaLens's one.
func (t *PoolTracker) getBookNaive(
	ctx context.Context, router string, token0, token1 common.Address,
) (feeBps uint16, epoch *big.Int, bidLeaves, askLeaves []decodedNode, blockNumber uint64, err error) {
	var feeConfig FeeConfigRPC
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI: deepstateV1ABI, Target: router, Method: methodFeeConfig,
	}, []any{&feeConfig})
	req.AddCall(&ethrpc.Call{
		ABI: deepstateV1ABI, Target: router, Method: methodPoolEpoch,
		Params: []any{poolID(token0, token1)},
	}, []any{&epoch})
	resp, err := req.TryAggregate()
	if err != nil {
		return 0, nil, nil, nil, 0, err
	}
	if !resp.Result[0] || !resp.Result[1] {
		return 0, nil, nil, nil, 0, ErrRPCCallReverted
	}

	book := bookID(token0, token1, epoch)

	var roots RootsRPC
	resp, err = t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI: deepstateV1ABI, Target: router, Method: methodRoots,
		Params: []any{token0, token1, epoch},
	}, []any{&roots}).TryAggregate()
	if err != nil {
		return 0, nil, nil, nil, 0, err
	}
	if !resp.Result[0] {
		return 0, nil, nil, nil, 0, ErrRPCCallReverted
	}

	bidLeaves, err = t.bfsWalkLeaves(ctx, router, book, roots.BidRoot)
	if err != nil {
		return 0, nil, nil, nil, 0, err
	}
	askLeaves, err = t.bfsWalkLeaves(ctx, router, book, roots.AskRoot)
	if err != nil {
		return 0, nil, nil, nil, 0, err
	}

	return feeConfig.Bps, epoch, bidLeaves, askLeaves, resp.BlockNumber.Uint64(), nil
}

// bfsWalkLeaves walks one side's radix tree from its root, level by level,
// batching all tree() reads at a level into one multicall. A node whose
// tree(id,node).leftNode is zero is a leaf; its own word (not the tree()
// result) is the packed order to decode.
func (t *PoolTracker) bfsWalkLeaves(
	ctx context.Context, router string, book [32]byte, root [32]byte,
) ([]decodedNode, error) {
	var zero [32]byte
	if root == zero {
		return nil, nil
	}

	leaves := make([]decodedNode, 0, 16)
	frontier := [][32]byte{root}
	visited := 0

	for len(frontier) > 0 {
		visited += len(frontier)
		if visited > maxBFSNodes {
			return nil, ErrBookTooLarge
		}

		results := make([]TreeRPC, len(frontier))
		req := t.ethrpcClient.NewRequest().SetContext(ctx)
		for i, node := range frontier {
			req.AddCall(&ethrpc.Call{
				ABI: deepstateV1ABI, Target: router, Method: methodTree,
				Params: []any{book, node},
			}, []any{&results[i]})
		}
		resp, err := req.TryAggregate()
		if err != nil {
			return nil, err
		}

		next := make([][32]byte, 0, len(frontier))
		for i, node := range frontier {
			if !resp.Result[i] {
				continue
			}
			if results[i].LeftNode == zero {
				leaves = append(leaves, decodeNode(node))
				continue
			}
			next = append(next, results[i].LeftNode, results[i].RightNode)
		}
		frontier = next
	}

	return leaves, nil
}

// buildLevels aggregates leaves by tick and converts them into the generic
// order-book package's flat, decimal-normalized [size, price] levels. See
// node.go's tickPrice and this package's price-derivation notes in
// context/deepstate/output/tracker.md for the token0/token1 <-> price
// direction proof.
func buildLevels(leaves []decodedNode, dec0, dec1 uint8, invert bool) []orderbook.Level {
	sums := make(map[int32]float64, len(leaves))
	for _, leaf := range leaves {
		qf := leaf.quantity.Float64()
		sums[leaf.tick] += qf / math.Pow10(int(dec0))
	}

	ticks := make([]int32, 0, len(sums))
	for tick := range sums {
		ticks = append(ticks, tick)
	}
	if invert {
		sort.Slice(ticks, func(i, j int) bool { return ticks[i] < ticks[j] }) // best ask = lowest price first
	} else {
		sort.Slice(ticks, func(i, j int) bool { return ticks[i] > ticks[j] }) // best bid = highest price first
	}

	levels := make([]orderbook.Level, 1, len(ticks)+1) // levels[0] = {0,0} dummy min-trade sentinel
	decAdj := math.Pow10(int(dec0) - int(dec1))
	for _, tick := range ticks {
		priceToken1PerToken0 := tickPrice(tick) * decAdj
		sizeToken0 := sums[tick]
		if invert {
			levels = append(levels, orderbook.Level{sizeToken0 * priceToken1PerToken0, 1 / priceToken1PerToken0})
		} else {
			levels = append(levels, orderbook.Level{sizeToken0, priceToken1PerToken0})
		}
	}
	return levels
}
