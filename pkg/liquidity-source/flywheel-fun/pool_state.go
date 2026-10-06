package flywheelfun

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	v4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

var stateABI = mustABI(`[
 {"type":"function","name":"extsload","inputs":[{"type":"bytes32[]"}],"outputs":[{"type":"bytes32[]"}]},
 {"type":"function","name":"poolIds","inputs":[{"type":"address"}],"outputs":[{"type":"bytes32"}]},
 {"type":"function","name":"poolKey","inputs":[],"outputs":[{"type":"tuple","components":[{"name":"currency0","type":"address"},{"name":"currency1","type":"address"},{"name":"fee","type":"uint24"},{"name":"tickSpacing","type":"int24"},{"name":"hooks","type":"address"}]}]},
 {"type":"function","name":"slot0","inputs":[],"outputs":[{"type":"uint160","name":"sqrtPriceX96"},{"type":"int24","name":"tick"},{"type":"uint16","name":"observationIndex"},{"type":"uint16","name":"observationCardinality"},{"type":"uint16","name":"observationCardinalityNext"},{"type":"uint8","name":"feeProtocol"},{"type":"bool","name":"unlocked"}]},
 {"type":"function","name":"token0","inputs":[],"outputs":[{"type":"address"}]},
 {"type":"function","name":"token1","inputs":[],"outputs":[{"type":"address"}]}
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
func poolEntity(id string, key poolKey, block *big.Int, extra v3.Extra, fee uint32) (entity.Pool, error) {
	data, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}
	static, err := json.Marshal(v4.StaticExtra{Fee: uint32(key.Fee.Uint64()), TickSpacing: int32(key.TickSpacing.Int64()), HooksAddress: key.Hooks, IsNative: [2]bool{key.Currency0 == (common.Address{}), key.Currency1 == (common.Address{})}})
	if err != nil {
		return entity.Pool{}, err
	}
	r0, r1 := v4.EstimateReservesFromTicks(extra.SqrtPriceX96, extra.Ticks)
	return entity.Pool{Address: id, Exchange: v4.DexType, Type: v4.DexType, BlockNumber: block.Uint64(), SwapFee: float64(fee), Extra: string(data), StaticExtra: string(static), Tokens: []*entity.PoolToken{{Address: asset(key.Currency0), Swappable: true}, {Address: asset(key.Currency1), Swappable: true}}, Reserves: entity.PoolReserves{r0.String(), r1.String()}}, nil
}

// trackV4 reads the launch's canonical, hook-locked V4 market.
func (t *PoolTracker) trackV4(ctx context.Context, key poolKey, block *big.Int) (entity.Pool, [2]uint32, error) {
	var protocol [2]uint32
	id, err := keyID(key)
	if err != nil {
		return entity.Pool{}, protocol, err
	}
	if key.Hooks != common.HexToAddress(NativeHook) || key.Fee.Uint64() != 0x800000 || key.TickSpacing.Int64() != 200 || key.Currency0 == (common.Address{}) {
		return entity.Pool{}, protocol, ErrState
	}
	root := mappingSlot(common.HexToHash(id).Big(), common.BigToHash(big.NewInt(6)))
	words, err := t.loadSlots(ctx, []common.Hash{root, addSlot(root, 3)}, block)
	if err != nil {
		return entity.Pool{}, protocol, err
	}
	slot := words[0].Big()
	mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 160), big.NewInt(1))
	price := new(big.Int).And(slot, mask)
	tick := signed(new(big.Int).Rsh(slot, 160), 24)
	packed := new(big.Int).Rsh(slot, 184).Uint64() & 0xffffff
	protocol = [2]uint32{uint32(packed & 4095), uint32(packed >> 12)}
	liquidity := new(big.Int).And(words[1].Big(), new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1)))
	if price.Sign() == 0 || liquidity.Sign() == 0 || protocol[0] > 1000 || protocol[1] > 1000 {
		return entity.Pool{}, protocol, ErrState
	}
	// The native hook pins one full-range position.
	indices := []int{-887200, 887200}
	slots := make([]common.Hash, len(indices))
	for i, ix := range indices {
		slots[i] = mappingSlot(big.NewInt(int64(ix)), addSlot(root, 4))
	}
	ticks, err := t.loadSlots(ctx, slots, block)
	if err != nil {
		return entity.Pool{}, protocol, err
	}
	extra := v3.Extra{Liquidity: liquidity, SqrtPriceX96: price, Tick: tick, TickSpacing: key.TickSpacing.Uint64()}
	mask128 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
	for i, w := range ticks {
		x := w.Big()
		gross := new(big.Int).And(x, mask128)
		net := signed(new(big.Int).Rsh(x, 128), 128)
		if gross.Sign() == 0 {
			return entity.Pool{}, protocol, ErrState
		}
		extra.Ticks = append(extra.Ticks, v3.Tick{Index: indices[i], LiquidityGross: gross, LiquidityNet: net})
	}
	if extra.Ticks[0].LiquidityNet.Cmp(liquidity) != 0 || new(big.Int).Neg(extra.Ticks[1].LiquidityNet).Cmp(liquidity) != 0 || extra.Ticks[0].LiquidityGross.Cmp(liquidity) != 0 || extra.Ticks[1].LiquidityGross.Cmp(liquidity) != 0 {
		return entity.Pool{}, protocol, ErrState
	}
	// The native hook charges no LP fee.
	p, err := poolEntity(id, key, block, extra, protocol[0])
	return p, protocol, err
}

func (t *PoolTracker) canonical(ctx context.Context, token string, c CurveRPC, block *big.Int) (entity.Pool, [2]uint32, error) {
	var wrapped struct{ Key poolKey }
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
	return t.trackV4(ctx, key, block)
}

type v3Slot struct {
	SqrtPriceX96, Tick                                                   *big.Int
	ObservationIndex, ObservationCardinality, ObservationCardinalityNext uint16
	FeeProtocol                                                          uint8
	Unlocked                                                             bool
}
