package btrprop

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

// poolABI is the embedded abis/Pool.json, a pinned copy of the deployed BTR Pool ABI.
var poolABI = lo.Must(abi.JSON(bytes.NewReader(poolABIJSONBytes)))

// registryABI covers the lane registry: the pool's `registry` getter (selector
// 0xbcf0196b; its source name is unknown, so the ID is pinned) and
// Registry.perms(caller), the lane bitmask swap_qe checks when SWAP_GATED is set.
var registryABI = func() abi.ABI {
	addrT, u256T := lo.Must(abi.NewType("address", "", nil)), lo.Must(abi.NewType("uint256", "", nil))
	reg := abi.NewMethod("registry", "registry", abi.Function, "view", false, false, nil, abi.Arguments{{Type: addrT}})
	reg.ID = []byte{0xbc, 0xf0, 0x19, 0x6b}
	perms := abi.NewMethod("perms", "perms", abi.Function, "view", false, false,
		abi.Arguments{{Name: "caller", Type: addrT}}, abi.Arguments{{Type: u256T}})
	return abi.ABI{Methods: map[string]abi.Method{"registry": reg, "perms": perms}}
}()
