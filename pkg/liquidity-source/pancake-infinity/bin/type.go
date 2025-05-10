package bin

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type SubgraphToken struct {
	ID       string `json:"id"`
	Decimals string `json:"decimals"`
}

type SubgraphPool struct {
	ID         string        `json:"id"`
	TokenX     SubgraphToken `json:"tokenX"`
	TokenY     SubgraphToken `json:"tokenY"`
	Hooks      string        `json:"hooks"`
	Parameters string        `json:"parameters"`
	Timestamp  string        `json:"timestamp"`
}

type StaticExtra struct {
	HasSwapPermissions bool           `json:"hsp"`
	IsNative           [2]bool        `json:"0x0"`
	Parameters         string         `json:"params"`
	BinStep            uint16         `json:"bs"`
	PoolManagerAddress common.Address `json:"pm"`
	HooksAddress       common.Address `json:"hooks"`
	Permit2Address     common.Address `json:"p2"`
	VaultAddress       common.Address `json:"vault"`
	Multicall3Address  common.Address `json:"m3"`
}

type Slot0Data struct {
	ActiveId    *big.Int `json:"activeId"`
	ProtocolFee *big.Int `json:"protocolFee"`
	LpFee       *big.Int `json:"lpFee"`
}

type FetchRPCResult struct {
	Slot0 Slot0Data `json:"slot0"`
}

type Extra struct {
	ProtocolFee *uint256.Int `json:"protocolFee"`
	ActiveBinID uint32       `json:"activeBinId"`
	Bins        []Bin        `json:"bins"`
}

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
}

type SwapInfo struct {
	NewActiveID        uint32              `json:"-"`
	BinsReserveChanges []binReserveChanges `json:"-"`
}

type Reserve struct {
	ReserveX *big.Int
	ReserveY *big.Int
}

type SubgraphBin struct {
	BinID    int32  `json:"binId"`
	ReserveX string `json:"reserveX"`
	ReserveY string `json:"reserveY"`
}

type LBPair struct {
	ID       string        `json:"id"`
	TokenX   SubgraphToken `json:"tokenX"`
	TokenY   SubgraphToken `json:"tokenY"`
	ReserveX string        `json:"reserveX"`
	ReserveY string        `json:"reserveY"`
	Bins     []SubgraphBin `json:"bins"`
}

type swapResult struct {
	Amount             *uint256.Int
	Fee                *uint256.Int
	NewActiveID        uint32
	BinsReserveChanges []binReserveChanges
}
