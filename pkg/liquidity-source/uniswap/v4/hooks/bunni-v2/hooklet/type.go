package hooklet

import (
	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

type HookletParams struct {
	RpcClient      *ethrpc.Client
	HookletAddress common.Address
	HookletExtra   uniswapv4.HookExtra
	PoolId         common.Hash
}

type SwapParams struct {
	ZeroForOne bool
}
