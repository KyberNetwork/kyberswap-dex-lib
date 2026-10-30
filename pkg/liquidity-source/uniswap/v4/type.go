package uniswapv4

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

type PoolSwapInfo = uniswapv3.SwapInfo

type SwapInfo struct {
	PoolSwapInfo
	HookSwapInfo any
	// FewReserve is the underlying balance left in the unwrapped Few token's contract after this
	// swap, nil when the swap unwraps nothing; FewIdx is that Few token's pool index.
	FewReserve *uint256.Int `json:"-"`
	FewIdx     int          `json:"-"`
}

type SubgraphToken struct {
	ID       string `json:"id"`
	Decimals string `json:"decimals"`
}

type SubgraphPool struct {
	ID                 string        `json:"id"`
	Token0             SubgraphToken `json:"token0"`
	Token1             SubgraphToken `json:"token1"`
	Fee                string        `json:"feeTier"`
	TickSpacing        string        `json:"tickSpacing"`
	Hooks              string        `json:"hooks"`
	CreatedAtTimestamp string        `json:"createdAtTimestamp"`
}

type StaticExtra struct {
	IsNative               [2]bool        `json:"0x0"`
	Fee                    uint32         `json:"fee"`
	TickSpacing            int32          `json:"tS"`
	HooksAddress           common.Address `json:"hooks"`
	UniversalRouterAddress common.Address `json:"uR"`
	Permit2Address         common.Address `json:"pm2"`
	Multicall3Address      common.Address `json:"mc3"`
	// PoolManager is set only for forks with their own PoolManager; nil means PoolManager[chainID].
	PoolManager *common.Address `json:"pM,omitempty"`
}

type Extra struct {
	*uniswapv3.Extra
	HookExtra json.RawMessage `json:"hX,omitempty"`
	// TrackedAt (lv) is the unix time the tracker last confirmed the pool's state, kept outside hX
	// so hX stays pure chain state. Only HookWithLiveness hooks carry it.
	TrackedAt int64 `json:"lv,omitempty"`
	// FewReserves (fwR), aligned with token order, is the underlying balance held by each
	// Few-wrapped pool token's contract: FewWrappedToken.unwrapTo pays out of that balance alone,
	// so it caps how much of the token can be unwrapped. Set only for pools holding a Few token.
	FewReserves []*big.Int `json:"fwR,omitempty"`
}

type ExtraU256 struct {
	*uniswapv3.ExtraTickU256
	HookExtra   json.RawMessage `json:"hX,omitempty"`
	TrackedAt   int64           `json:"lv,omitempty"`
	FewReserves []*uint256.Int  `json:"fwR,omitempty"`
}

type Slot0Data struct {
	SqrtPriceX96 *big.Int `json:"sqrtPriceX96"`
	Tick         *big.Int `json:"tick"`
	ProtocolFee  uint32   `json:"protocolFee"`
	LpFee        uint32   `json:"lpFee"`
}

type FetchRPCResult struct {
	Liquidity   *big.Int            `json:"l"`
	Slot0       Slot0Data           `json:"s0"`
	TickSpacing int32               `json:"tS"`
	Reserves    entity.PoolReserves `json:"rs"`
	SwapFee     uint32              `json:"swapFee"`
	HookExtra   json.RawMessage     `json:"hX,omitempty"`
	TrackedAt   int64               `json:"lv,omitempty"`
	FewReserves []*big.Int          `json:"fwR,omitempty"`
}

// ToExtra builds the pool's Extra from this fetch and the given ticks. Every
// writer of the tracked Extra goes through it, so a field added here reaches
// all of them.
func (r *FetchRPCResult) ToExtra(ticks []Tick) Extra {
	return Extra{
		Extra: &uniswapv3.Extra{
			Liquidity:    r.Liquidity,
			TickSpacing:  uint64(r.TickSpacing),
			SqrtPriceX96: r.Slot0.SqrtPriceX96,
			Tick:         r.Slot0.Tick,
			Ticks:        ticks,
		},
		HookExtra:   r.HookExtra,
		TrackedAt:   r.TrackedAt,
		FewReserves: r.FewReserves,
	}
}

type TicksResp struct {
	LiquidityGross *big.Int
	LiquidityNet   *big.Int
}

type Tick = uniswapv3.Tick

type PoolMetaInfo struct {
	BlockNumber uint64         `json:"blockNumber"`
	Router      common.Address `json:"router"`
	Permit2Addr common.Address `json:"permit2Addr"`
	TokenIn     common.Address `json:"tokenIn"`
	TokenOut    common.Address `json:"tokenOut"`
	Fee         uint32         `json:"fee"`
	TickSpacing int32          `json:"tickSpacing"`
	HookAddress common.Address `json:"hookAddress"`
	HookData    []byte         `json:"hookData"`
	PriceLimit  *uint256.Int   `json:"priceLimit"`
	// PoolManager is the pool's manager (own, else PoolManager[chainID]); omitted when unknown.
	PoolManager *common.Address `json:"manager,omitempty"`

	TokenWrapMetadata *TokenWrapMetadata `json:"tokenWrapMetadata,omitempty"`
}

type TokenWrapMetadata struct {
	WrapInfo   *WrapInfo `json:"wrapInfo,omitempty"`
	UnwrapInfo *WrapInfo `json:"unwrapInfo,omitempty"`
}

func (t *TokenWrapMetadata) ShouldWrap() bool {
	return t != nil && t.WrapInfo != nil
}

func (t *TokenWrapMetadata) ShouldUnwrap() bool {
	return t != nil && t.UnwrapInfo != nil
}

type WrapInfo struct {
	TokenIn     string `json:"tokenIn,omitempty"`
	TokenOut    string `json:"tokenOut,omitempty"`
	HookAddress string `json:"hookAddress,omitempty"`
	PoolAddress string `json:"poolAddress,omitempty"`
	TickSpacing int32  `json:"tickSpacing,omitempty"`
	Fee         uint32 `json:"fee,omitempty"`
	HookData    []byte `json:"hookData,omitempty"`
}

type PoolKey struct {
	Currency0   common.Address
	Currency1   common.Address
	Fee         *big.Int
	TickSpacing *big.Int
	Hooks       common.Address
}
