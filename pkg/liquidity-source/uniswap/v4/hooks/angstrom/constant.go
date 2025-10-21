package angstrom

import "github.com/ethereum/go-ethereum/common"

const Handler = "uniswap-v4-angstrom"

const GET_ATTESTATIONS_PATH = "/getAttestations"

var Adapter = common.HexToAddress("0xb535aeb27335b91e1b5bccbd64888ba7574efbf8")

var HookAddresses = []common.Address{
	common.HexToAddress("0x0000000aa232009084bd71a5797d089aa4edfad4"),
}
