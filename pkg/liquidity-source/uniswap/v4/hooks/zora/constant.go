package zora

import "github.com/ethereum/go-ethereum/common"

var (
	HookAddresses = []common.Address{
		// CreatorCoinHook
		common.HexToAddress("0xa1ebdd5ca6470bbd67114331387f2dda7bfad040"),
		common.HexToAddress("0x5e5d19d22c85a4aef7c1fdf25fb22a5a38f71040"),
		common.HexToAddress("0xd61a675f8a0c67a73dc3b54fb7318b4d91409040"),
		common.HexToAddress("0x9278f6e55ce58519c79dc1ab0ad3b29ea7821040"),
		common.HexToAddress("0x8218FA8d7922e22aED3556a09D5A715F16Ad5040"),
		common.HexToAddress("0x1258e5f3C71ca9dCE95Ce734Ba5759532E46D040"),

		// ContentCoinHook
		// common.HexToAddress("0xfff800b76768da8ab6aab527021e4a6a91219040"), // disabled for now: large number of pools
		common.HexToAddress("0x5bf219b3cc11e3f6dd8dc8fc89d7d1deb0431040"),
		common.HexToAddress("0x9ea932730a7787000042e34390b8e435dd839040"),
	}

	ZoraAddress = common.HexToAddress("0x1111111111166b7FE7bd91427724B487980aFc69")
)

func IsZora(token common.Address) bool {
	return token.Cmp(ZoraAddress) == 0
}
