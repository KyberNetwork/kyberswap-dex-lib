package peach

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType             = valueobject.ExchangeUniswapV4Peach
	feePrecision uint64 = 1_000_000
	// Includes the callback, observation write and bounded math-module call. See README.md.
	gasBeforeSwap int64 = 70_000
	gasMidBlock   int64 = 220_000
)

var (
	ErrUntracked    = errors.New("peach: pool is not tracked")
	ErrDeployment   = errors.New("peach: unsupported chain or hook")
	ErrInvalidState = errors.New("peach: invalid fee state")
	ErrPaused       = errors.New("peach: hook or pool paused")
	ErrInvalidPrice = errors.New("peach: invalid sqrt price")
)

type deployment struct {
	ChainID valueobject.ChainID
	RWA     bool
}

var deployments = map[common.Address]deployment{
	common.HexToAddress("0x4D1546682BC8252E52c1c1820F0AaA7694C9e080"): {56, false},
	common.HexToAddress("0x1e4605837970EeF394662B1a9e88e09292636080"): {56, false},
	common.HexToAddress("0x98C819a522b91f76bAEAf499F3801e76890F2080"): {56, true},
	common.HexToAddress("0x8c960De62a26A688B72e6cbf644F46715642a080"): {5042, false},
	common.HexToAddress("0x3722d2840bFA4E9610d265DA08078d178bF46080"): {5042, true},
}

func hookAddresses() []common.Address {
	addresses := make([]common.Address, 0, len(deployments))
	for address := range deployments {
		addresses = append(addresses, address)
	}
	return addresses
}
