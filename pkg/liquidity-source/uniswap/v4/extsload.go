package uniswapv4

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/ethrpc"
)

// With no StateViewAddress, pool state is read straight from the PoolManager: StateView is only a
// wrapper over extsload, so forks without a StateView need nothing deployed.
//
// Pool.State layout (v4-core): slot0, feeGrowthGlobal0, feeGrowthGlobal1, liquidity (+3),
// ticks mapping (+4); the pools mapping sits at slot 6 of the PoolManager.
const (
	poolsMappingSlot = 6
	liquidityOffset  = 3
	ticksMapOffset   = 4
)

func (t *PoolTracker) useExtsload() bool { return t.config.StateViewAddress == "" }

func (t *PoolTracker) managerAddress() string {
	if pm := t.config.poolManager(); pm != nil {
		return pm.Hex()
	}
	return PoolManager[t.config.ChainID].Hex()
}

func (t *PoolTracker) extsloadCall(slot common.Hash) *ethrpc.Call {
	return &ethrpc.Call{ABI: poolManagerABI, Target: t.managerAddress(), Method: "extsload", Params: []any{slot}}
}

// stateSlot is the storage slot of Pool.State for poolID: keccak256(poolId . pools slot).
func stateSlot(poolID common.Hash) common.Hash {
	var slot [32]byte
	slot[31] = poolsMappingSlot
	return crypto.Keccak256Hash(poolID[:], slot[:])
}

func addOffset(h common.Hash, off uint64) common.Hash {
	var v uint256.Int
	v.SetBytes(h[:])
	v.AddUint64(&v, off)
	return v.Bytes32()
}

// tickSlot is the storage slot of ticks[tick] (Pool.TickInfo) for the pool whose state starts at base.
func tickSlot(base common.Hash, tick int64) common.Hash {
	var key uint256.Int // int24 -> sign-extended 256-bit key
	key.SetUint64(uint64(tick))
	if tick < 0 {
		key.SetAllOne()
		key.Sub(&key, new(uint256.Int).SetUint64(uint64(-tick)-1))
	}
	k := key.Bytes32()
	m := addOffset(base, ticksMapOffset)
	return crypto.Keccak256Hash(k[:], m[:])
}

// decodeSlot0 unpacks Slot0: sqrtPriceX96 (160) | tick (24, signed) | protocolFee (24) | lpFee (24).
func decodeSlot0(w [32]byte) Slot0Data {
	var v, sqrt uint256.Int
	v.SetBytes(w[:])
	sqrt.SetBytes(w[12:]) // low 160 bits
	rest := new(uint256.Int).Rsh(&v, 160).Uint64()
	tick := int64(rest & 0xffffff)
	if tick >= 1<<23 {
		tick -= 1 << 24
	}
	return Slot0Data{
		SqrtPriceX96: sqrt.ToBig(),
		Tick:         big.NewInt(tick),
		ProtocolFee:  uint32(rest >> 24 & 0xffffff),
		LpFee:        uint32(rest >> 48 & 0xffffff),
	}
}

// decodeLiquidity reads the low uint128 of a word.
func decodeLiquidity(w [32]byte) *big.Int { return new(big.Int).SetBytes(w[16:]) }

// decodeTickInfo reads TickInfo's first word: liquidityGross (low uint128) | liquidityNet (high int128).
func decodeTickInfo(w [32]byte) (gross, net *big.Int) {
	gross = new(big.Int).SetBytes(w[16:])
	net = new(big.Int).SetBytes(w[:16])
	if w[0]&0x80 != 0 {
		net.Sub(net, new(big.Int).Lsh(big.NewInt(1), 128))
	}
	return gross, net
}
