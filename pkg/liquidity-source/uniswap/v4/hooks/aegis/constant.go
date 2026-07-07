package aegis

import (
	"github.com/ethereum/go-ethereum/common"
)

var HookAddresses = []common.Address{
	common.HexToAddress("0xA0b0D2d00fD544D8E0887F1a3cEDd6e24Baf10cc"), // unichain 1.0
	common.HexToAddress("0xb4f4949e8D0a177bb6D2fea33e9516Bb219610cc"), // monad 1.0
	common.HexToAddress("0x8f29BD5c8429730fA4C46e6295c4e679eDEdd0cC"), // ethereum/base 1.1
	common.HexToAddress("0xe449E013004DB4a5681E9622ca10c5Ba0eA610cc"), // monad 1.1
}
