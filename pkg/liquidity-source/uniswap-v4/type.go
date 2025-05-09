package uniswapv4

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswapv3"
)

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
}

type Extra = uniswapv3.Extra
type ExtraTickU256 = uniswapv3.ExtraTickU256

type Slot0Data struct {
	SqrtPriceX96 *big.Int `json:"sqrtPriceX96"`
	Tick         *big.Int `json:"tick"`
	ProtocolFee  *big.Int `json:"protocolFee"`
	LpFee        *big.Int `json:"lpFee"`
}

type FetchRPCResult struct {
	Liquidity   *big.Int  `json:"liquidity"`
	Slot0       Slot0Data `json:"slot0"`
	TickSpacing int32     `json:"tickSpacing"`
}

type Tick = uniswapv3.Tick

type PoolMetaInfo struct {
	Router      common.Address `json:"router"`
	Permit2Addr common.Address `json:"permit2Addr"`
	TokenIn     common.Address `json:"tokenIn"`
	TokenOut    common.Address `json:"tokenOut"`
	Fee         uint32         `json:"fee"`
	TickSpacing int32          `json:"tickSpacing"`
	HookAddress common.Address `json:"hookAddress"`
	HookData    []byte         `json:"hookData"`
	PriceLimit  *uint256.Int   `json:"priceLimit"`
	ApprovalAddress string         `json:"approvalAddress"`
}
