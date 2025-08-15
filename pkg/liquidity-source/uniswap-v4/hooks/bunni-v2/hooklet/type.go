package hooklet

import (
	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
)

type HookletParams struct {
	RpcClient      *ethrpc.Client
	HookletAddress common.Address
	HookletExtra   string
	PoolId         common.Hash
}

type SwapParams struct {
	ZeroForOne bool
}
