package ozlimitorder

import "github.com/ethereum/go-ethereum/common"

// HookAddresses are deployments of OpenZeppelin Uniswap Hooks' LimitOrderHook
// (https://github.com/OpenZeppelin/uniswap-hooks/blob/b5f10574f943911635754dd76ed6ce95858b7d93/src/general/LimitOrderHook.sol, versioned 1.3.0), unmodified.
var HookAddresses = []common.Address{
	// Monad (143): CREATE2 via 0x4e59b44847b379578588920cA78FbF26c0B4956C, PoolManager 0x188d586Ddcf52439676Ca21A244753fA19F9Ea8e
	common.HexToAddress("0x94Df2bffA788BBd918179f4CCB5A22DbAD7e1040"),
}
