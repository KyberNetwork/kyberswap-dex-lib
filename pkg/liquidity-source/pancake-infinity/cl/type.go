package cl

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
	Hooks              string        `json:"hooks"`
	Parameters         string        `json:"parameters"`
	CreatedAtTimestamp string        `json:"createdAtTimestamp"`
}

type StaticExtra struct {
	HasSwapPermissions bool           `json:"hsp"`
	IsNative           [2]bool        `json:"0x0"`
	Fee                uint32         `json:"fee"`
	Parameters         string         `json:"params"`
	TickSpacing        uint64         `json:"tS"`
	PoolManagerAddress common.Address `json:"pm"`
	HooksAddress       common.Address `json:"hooks"`
	Permit2Address     common.Address `json:"p2"`
	VaultAddress       common.Address `json:"vault"`
	Multicall3Address  common.Address `json:"m3"`
}

type Extra = uniswapv3.Extra

type Slot0Data struct {
	SqrtPriceX96 *big.Int `json:"sqrtPriceX96"`
	Tick         *big.Int `json:"tick"`
	ProtocolFee  uint32   `json:"protocolFee"`
	LpFee        uint32   `json:"lpFee"`
}

type FetchRPCResult struct {
	Liquidity   *big.Int  `json:"liquidity"`
	Slot0       Slot0Data `json:"slot0"`
	TickSpacing uint64    `json:"tickSpacing"`
	SwapFee     uint32    `json:"swapFee"`
}

type Tick = uniswapv3.Tick

type PoolMetaInfo struct {
	Vault       common.Address `json:"vault"`
	PoolManager common.Address `json:"poolManager"`
	Permit2Addr common.Address `json:"permit2Addr"`
	TokenIn     common.Address `json:"tokenIn"`
	TokenOut    common.Address `json:"tokenOut"`
	Fee         uint32         `json:"fee"`
	Parameters  string         `json:"parameters"`
	HookAddress common.Address `json:"hookAddress"`
	HookData    []byte         `json:"hookData"`
	PriceLimit  *uint256.Int   `json:"priceLimit"`
}
