package flywheelfun

import (
	"context"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	v4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// Many launches share BOOMER's route. One tracker snapshot per route/block avoids
// repeating the same tick scan for every launch; the block hash isolates reorgs.
func (t *PoolTracker) cachedRoute(ctx context.Context, h RouteHop, block *big.Int, hash common.Hash) (entity.Pool, RouteHop, error) {
	id, err := h.id()
	if err != nil {
		return entity.Pool{}, h, err
	}
	t.routeMu.Lock()
	defer t.routeMu.Unlock()
	if t.routeBlock != hash {
		t.routeBlock = hash
		t.routeCache = make(map[string]routeSnapshot)
	}
	if v, ok := t.routeCache[id]; ok {
		return v.pool, v.hop, nil
	}
	var p entity.Pool
	if h.Kind == 4 {
		p, h, err = t.trackV4(ctx, h, block, false)
	} else if h.Kind == 3 {
		p, h, err = t.trackV3(ctx, h, block)
	} else {
		return p, h, ErrUnsupported
	}
	if err == nil {
		if len(t.routeCache) >= 64 {
			t.routeCache = make(map[string]routeSnapshot)
		}
		t.routeCache[id] = routeSnapshot{p, h}
	}
	return p, h, err
}

var stateABI = mustABI(`[
 {"type":"function","name":"extsload","inputs":[{"type":"bytes32[]"}],"outputs":[{"type":"bytes32[]"}]},
 {"type":"function","name":"poolIds","inputs":[{"type":"address"}],"outputs":[{"type":"bytes32"}]},
 {"type":"function","name":"poolKey","inputs":[],"outputs":[{"type":"tuple","components":[{"name":"currency0","type":"address"},{"name":"currency1","type":"address"},{"name":"fee","type":"uint24"},{"name":"tickSpacing","type":"int24"},{"name":"hooks","type":"address"}]}]},
 {"type":"function","name":"slot0","inputs":[],"outputs":[{"type":"uint160","name":"sqrtPriceX96"},{"type":"int24","name":"tick"},{"type":"uint16","name":"observationIndex"},{"type":"uint16","name":"observationCardinality"},{"type":"uint16","name":"observationCardinalityNext"},{"type":"uint8","name":"feeProtocol"},{"type":"bool","name":"unlocked"}]},
 {"type":"function","name":"liquidity","inputs":[],"outputs":[{"type":"uint128"}]},
 {"type":"function","name":"tickSpacing","inputs":[],"outputs":[{"type":"int24"}]},
 {"type":"function","name":"factory","inputs":[],"outputs":[{"type":"address"}]},
 {"type":"function","name":"token0","inputs":[],"outputs":[{"type":"address"}]},
 {"type":"function","name":"token1","inputs":[],"outputs":[{"type":"address"}]},
 {"type":"function","name":"fee","inputs":[],"outputs":[{"type":"uint24"}]},
 {"type":"function","name":"getPool","inputs":[{"type":"address"},{"type":"address"},{"type":"uint24"}],"outputs":[{"type":"address"}]},
 {"type":"function","name":"tickBitmap","inputs":[{"type":"int16"}],"outputs":[{"type":"uint256"}]},
 {"type":"function","name":"ticks","inputs":[{"type":"int24"}],"outputs":[{"type":"uint128","name":"liquidityGross"},{"type":"int128","name":"liquidityNet"},{"type":"uint256","name":"feeGrowthOutside0X128"},{"type":"uint256","name":"feeGrowthOutside1X128"},{"type":"int56","name":"tickCumulativeOutside"},{"type":"uint160","name":"secondsPerLiquidityOutsideX128"},{"type":"uint32","name":"secondsOutside"},{"type":"bool","name":"initialized"}]}
]`)

func addSlot(slot common.Hash, n int64) common.Hash {
	return common.BigToHash(new(big.Int).Add(slot.Big(), big.NewInt(n)))
}
func mappingSlot(index *big.Int, slot common.Hash) common.Hash {
	x := new(big.Int).Set(index)
	if x.Sign() < 0 {
		x.Add(x, new(big.Int).Lsh(big.NewInt(1), 256))
	}
	return crypto.Keccak256Hash(common.LeftPadBytes(x.Bytes(), 32), slot[:])
}
func signed(n *big.Int, bits uint) *big.Int {
	out := new(big.Int).And(n, new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), bits), big.NewInt(1)))
	if out.Bit(int(bits-1)) != 0 {
		out.Sub(out, new(big.Int).Lsh(big.NewInt(1), bits))
	}
	return out
}
func (t *PoolTracker) loadSlots(ctx context.Context, slots []common.Hash, block *big.Int) ([]common.Hash, error) {
	var out []common.Hash
	if _, err := t.rpc.NewRequest().SetContext(ctx).SetBlockNumber(block).AddCall(&ethrpc.Call{ABI: stateABI, Target: Manager, Method: "extsload", Params: []any{slots}}, []any{&out}).Aggregate(); err != nil {
		return nil, err
	}
	if len(out) != len(slots) {
		return nil, ErrState
	}
	return out, nil
}
func wordRange(spacing int) (int, int, error) {
	if spacing < 1 || spacing > 32767 {
		return 0, 0, ErrState
	}
	// Bound initial full scans. Small-spacing pools require Kyber's incremental
	// tick indexer rather than an unbounded scan on every Flywheel refresh.
	lo := (-887272 / spacing) >> 8
	hi := (887272 / spacing) >> 8
	if hi-lo+1 > 256 {
		return 0, 0, ErrUnsupported
	}
	return lo, hi, nil
}
func poolEntity(id string, key PoolKey, block *big.Int, extra v3.Extra, fee uint32, kind uint8) (entity.Pool, error) {
	data, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}
	r0, r1 := v4.EstimateReservesFromTicks(extra.SqrtPriceX96, extra.Ticks)
	p := entity.Pool{Address: id, Exchange: v4.DexType, Type: v4.DexType, BlockNumber: block.Uint64(), SwapFee: float64(fee), Extra: string(data), Tokens: []*entity.PoolToken{{Address: asset(key.Currency0), Swappable: true}, {Address: asset(key.Currency1), Swappable: true}}, Reserves: entity.PoolReserves{r0.String(), r1.String()}}
	if kind == 3 {
		p.Type = v3.DexTypeUniswapV3
		p.Exchange = p.Type
	} else {
		b, e := json.Marshal(v4.StaticExtra{Fee: uint32(key.Fee.Uint64()), TickSpacing: int32(key.TickSpacing.Int64()), HooksAddress: key.Hooks, IsNative: [2]bool{key.Currency0 == (common.Address{}), key.Currency1 == (common.Address{})}})
		if e != nil {
			return p, e
		}
		p.StaticExtra = string(b)
	}
	return p, nil
}
func (t *PoolTracker) trackV4(ctx context.Context, h RouteHop, block *big.Int, canonical bool) (entity.Pool, RouteHop, error) {
	id, err := keyID(h.Key)
	if err != nil {
		return entity.Pool{}, h, err
	}
	if canonical {
		if h.Key.Hooks != common.HexToAddress(NativeHook) || h.Key.Fee.Uint64() != 0x800000 || h.Key.TickSpacing.Int64() != 200 || h.Key.Currency0 == (common.Address{}) {
			return entity.Pool{}, h, ErrState
		}
	} else if h.Key.Hooks != (common.Address{}) || h.Key.Fee.Uint64() >= 1_000_000 {
		return entity.Pool{}, h, ErrUnsupported
	}
	root := mappingSlot(common.HexToHash(id).Big(), common.BigToHash(big.NewInt(6)))
	words, err := t.loadSlots(ctx, []common.Hash{root, addSlot(root, 3)}, block)
	if err != nil {
		return entity.Pool{}, h, err
	}
	slot := words[0].Big()
	mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 160), big.NewInt(1))
	price := new(big.Int).And(slot, mask)
	tick := signed(new(big.Int).Rsh(slot, 160), 24)
	packed := new(big.Int).Rsh(slot, 184).Uint64() & 0xffffff
	lp := uint32(new(big.Int).Rsh(slot, 208).Uint64() & 0xffffff)
	h.Protocol = [2]uint32{uint32(packed & 4095), uint32(packed >> 12)}
	h.LPFee = lp
	liquidity := new(big.Int).And(words[1].Big(), new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1)))
	if price.Sign() == 0 || liquidity.Sign() == 0 || h.Protocol[0] > 1000 || h.Protocol[1] > 1000 || (!canonical && lp != uint32(h.Key.Fee.Uint64())) {
		return entity.Pool{}, h, ErrState
	}
	indices := []int{-887200, 887200}
	if !canonical {
		lo, hi, e := wordRange(int(h.Key.TickSpacing.Int64()))
		if e != nil {
			return entity.Pool{}, h, e
		}
		slots := make([]common.Hash, hi-lo+1)
		for w := lo; w <= hi; w++ {
			slots[w-lo] = mappingSlot(big.NewInt(int64(w)), addSlot(root, 5))
		}
		bitmaps, e := t.loadSlots(ctx, slots, block)
		if e != nil {
			return entity.Pool{}, h, e
		}
		indices = nil
		for j, b := range bitmaps {
			bits := b.Big()
			for bit := 0; bit < 256; bit++ {
				if bits.Bit(bit) != 0 {
					ix := ((lo+j)*256 + bit) * int(h.Key.TickSpacing.Int64())
					if ix < -887272 || ix > 887272 {
						return entity.Pool{}, h, ErrState
					}
					indices = append(indices, ix)
				}
			}
		}
	}
	if len(indices) < 2 || len(indices) > 1024 {
		return entity.Pool{}, h, ErrUnsupported
	}
	slots := make([]common.Hash, len(indices))
	for i, ix := range indices {
		slots[i] = mappingSlot(big.NewInt(int64(ix)), addSlot(root, 4))
	}
	ticks, err := t.loadSlots(ctx, slots, block)
	if err != nil {
		return entity.Pool{}, h, err
	}
	extra := v3.Extra{Liquidity: liquidity, SqrtPriceX96: price, Tick: tick, TickSpacing: uint64(h.Key.TickSpacing.Uint64())}
	mask128 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
	for i, w := range ticks {
		x := w.Big()
		gross := new(big.Int).And(x, mask128)
		net := signed(new(big.Int).Rsh(x, 128), 128)
		if gross.Sign() == 0 {
			return entity.Pool{}, h, ErrState
		}
		extra.Ticks = append(extra.Ticks, v3.Tick{Index: indices[i], LiquidityGross: gross, LiquidityNet: net})
	}
	if canonical {
		if extra.Ticks[0].LiquidityNet.Cmp(liquidity) != 0 || new(big.Int).Neg(extra.Ticks[1].LiquidityNet).Cmp(liquidity) != 0 || extra.Ticks[0].LiquidityGross.Cmp(liquidity) != 0 || extra.Ticks[1].LiquidityGross.Cmp(liquidity) != 0 {
			return entity.Pool{}, h, ErrState
		}
		lp = 0
	}
	combined := h.Protocol[0] + lp - uint32(uint64(h.Protocol[0])*uint64(lp)/1_000_000)
	p, err := poolEntity(id, h.Key, block, extra, combined, 4)
	return p, h, err
}

func (t *PoolTracker) canonical(ctx context.Context, token string, c CurveRPC, block *big.Int) (entity.Pool, [2]uint32, error) {
	var wrapped struct{ Key PoolKey }
	var id common.Hash
	if c.Pool == (common.Address{}) || c.PositionTokenId == nil || c.PositionTokenId.Sign() == 0 {
		return entity.Pool{}, [2]uint32{}, ErrState
	}
	_, err := t.rpc.NewRequest().SetContext(ctx).SetBlockNumber(block).
		AddCall(&ethrpc.Call{ABI: stateABI, Target: c.Pool.Hex(), Method: "poolKey"}, []any{&wrapped}).
		AddCall(&ethrpc.Call{ABI: stateABI, Target: Factory, Method: "poolIds", Params: []any{common.HexToAddress(token)}}, []any{&id}).Aggregate()
	if err != nil {
		return entity.Pool{}, [2]uint32{}, err
	}
	key := wrapped.Key
	computed, err := keyID(key)
	if err != nil || computed != id.Hex() || !((key.Currency0 == common.HexToAddress(token) && key.Currency1 == c.QuoteAsset) || (key.Currency1 == common.HexToAddress(token) && key.Currency0 == c.QuoteAsset)) {
		return entity.Pool{}, [2]uint32{}, ErrState
	}
	p, h, err := t.trackV4(ctx, RouteHop{Kind: 4, Key: key}, block, true)
	return p, h.Protocol, err
}

type v3Slot struct {
	SqrtPriceX96, Tick                                                   *big.Int
	ObservationIndex, ObservationCardinality, ObservationCardinalityNext uint16
	FeeProtocol                                                          uint8
	Unlocked                                                             bool
}

func (t *PoolTracker) trackV3(ctx context.Context, h RouteHop, block *big.Int) (entity.Pool, RouteHop, error) {
	id, err := h.id()
	if err != nil {
		return entity.Pool{}, h, err
	}
	var f, a, b common.Address
	var fee, spacing, liq *big.Int
	var slot v3Slot
	req := t.rpc.NewRequest().SetContext(ctx).SetBlockNumber(block)
	for _, read := range []struct {
		method string
		out    any
	}{{"factory", &f}, {"token0", &a}, {"token1", &b}, {"fee", &fee}, {"tickSpacing", &spacing}, {"liquidity", &liq}, {"slot0", &slot}} {
		req.AddCall(&ethrpc.Call{ABI: stateABI, Target: id, Method: read.method}, []any{read.out})
	}
	if _, err = req.Aggregate(); err != nil {
		return entity.Pool{}, h, err
	}
	if f != common.HexToAddress(V3Factory) || a == (common.Address{}) || a.Big().Cmp(b.Big()) >= 0 || fee == nil || !fee.IsUint64() || fee.Uint64() >= 1_000_000 || spacing == nil || !spacing.IsInt64() || liq == nil || liq.Sign() <= 0 || !slot.Unlocked {
		return entity.Pool{}, h, ErrState
	}
	lo, hi, err := wordRange(int(spacing.Int64()))
	if err != nil {
		return entity.Pool{}, h, err
	}
	var verified common.Address
	bitmaps := make([]*big.Int, hi-lo+1)
	req = t.rpc.NewRequest().SetContext(ctx).SetBlockNumber(block)
	req.AddCall(&ethrpc.Call{ABI: stateABI, Target: V3Factory, Method: "getPool", Params: []any{a, b, fee}}, []any{&verified})
	for w := lo; w <= hi; w++ {
		req.AddCall(&ethrpc.Call{ABI: stateABI, Target: id, Method: "tickBitmap", Params: []any{int16(w)}}, []any{&bitmaps[w-lo]})
	}
	if _, err = req.Aggregate(); err != nil {
		return entity.Pool{}, h, err
	}
	if verified != h.Pool {
		return entity.Pool{}, h, ErrState
	}
	var indices []int
	for j, bits := range bitmaps {
		if bits == nil {
			return entity.Pool{}, h, ErrState
		}
		for bit := 0; bit < 256; bit++ {
			if bits.Bit(bit) != 0 {
				ix := ((lo+j)*256 + bit) * int(spacing.Int64())
				if ix < -887272 || ix > 887272 {
					return entity.Pool{}, h, ErrState
				}
				indices = append(indices, ix)
			}
		}
	}
	if len(indices) < 2 || len(indices) > 1024 {
		return entity.Pool{}, h, ErrUnsupported
	}
	// The authenticated, immutable standard V3 factory uses ticks at slot 5.
	// Only its first packed word is needed (gross/net liquidity), not the three
	// additional fee/oracle words read by the public ticks() getter.
	// https://github.com/Uniswap/v3-core/blob/v1.0.0/contracts/UniswapV3Pool.sol#L67-L90
	ticks := make([]hexutil.Bytes, len(indices))
	for start := 0; start < len(indices); start += 100 {
		end := min(start+100, len(indices))
		batch := make([]rpc.BatchElem, end-start)
		for i := start; i < end; i++ {
			key := mappingSlot(big.NewInt(int64(indices[i])), common.BigToHash(big.NewInt(5)))
			batch[i-start] = rpc.BatchElem{Method: "eth_getStorageAt", Args: []any{h.Pool, key, hexutil.EncodeBig(block)}, Result: &ticks[i]}
		}
		if err = t.rpc.GetETHClient().Client().BatchCallContext(ctx, batch); err != nil {
			return entity.Pool{}, h, err
		}
		for _, item := range batch {
			if item.Error != nil {
				return entity.Pool{}, h, item.Error
			}
		}
	}
	extra := v3.Extra{Liquidity: liq, SqrtPriceX96: slot.SqrtPriceX96, Tick: slot.Tick, TickSpacing: spacing.Uint64()}
	for i, v := range ticks {
		if len(v) != 32 {
			return entity.Pool{}, h, ErrState
		}
		gross := new(big.Int).SetBytes(v[16:])
		net := signed(new(big.Int).SetBytes(v[:16]), 128)
		if gross.Sign() == 0 {
			return entity.Pool{}, h, ErrState
		}
		extra.Ticks = append(extra.Ticks, v3.Tick{Index: indices[i], LiquidityGross: gross, LiquidityNet: net})
	}
	h.Key = PoolKey{Currency0: a, Currency1: b, Fee: fee, TickSpacing: spacing}
	h.LPFee = uint32(fee.Uint64())
	p, err := poolEntity(strings.ToLower(id), h.Key, block, extra, uint32(fee.Uint64()), 3)
	return p, h, err
}
