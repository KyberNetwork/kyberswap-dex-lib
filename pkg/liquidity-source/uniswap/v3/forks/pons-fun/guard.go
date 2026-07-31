package ponsfun

import (
	"math/big"

	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Guard struct {
	launchToken         string
	multicall3          string
	restrictionEndBlock *big.Int // L1 block
	l1BlockNumber       *big.Int
}

func NewGuard(chainID valueobject.ChainID, multicall3, token0, token1 string) *Guard {
	var launchToken string
	switch {
	case valueobject.IsWrappedNative(token0, chainID):
		launchToken = token1
	case valueobject.IsWrappedNative(token1, chainID):
		launchToken = token0
	default:
		return nil
	}

	return &Guard{launchToken: launchToken, multicall3: multicall3}
}

func (g *Guard) AddCalls(req *ethrpc.Request) {
	if g == nil {
		return
	}

	req.AddCall(&ethrpc.Call{
		ABI:    ponsTokenABI,
		Target: g.launchToken,
		Method: "restrictionEndBlock",
	}, []any{&g.restrictionEndBlock}).AddCall(&ethrpc.Call{
		ABI:    abi.Multicall3ABI,
		Target: g.multicall3,
		Method: abi.Multicall3GetBlockNumber,
	}, []any{&g.l1BlockNumber})
}

func (g *Guard) BuyRestrictedToken() string {
	if g == nil {
		return ""
	}

	if g.l1BlockNumber == nil || g.restrictionEndBlock == nil ||
		g.l1BlockNumber.Uint64() > g.restrictionEndBlock.Uint64() {
		return ""
	}

	return g.launchToken
}
