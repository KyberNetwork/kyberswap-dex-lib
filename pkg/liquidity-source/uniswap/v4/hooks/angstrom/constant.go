package angstrom

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const Handler = "uniswap-v4-angstrom"

const GET_ATTESTATIONS_PATH = "/getAttestations"

// UnlockedFee/ProtocolUnlockedFee used in before/after swap hooks.
// It difficult to track these fields, since contract doesn't expose them.
// Only returns value from events, which can be seen from Angstrom Controller:
// https://etherscan.io/address/0x1746484EA5e11C75e009252c102C8C33e0315fD4#events
const UnlockedFee = 338

var ProtocolUnlockedFee = big.NewInt(112)
var ONE_E6 = big.NewInt(1_000_000)

var Adapter = common.HexToAddress("0xb535aeb27335b91e1b5bccbd64888ba7574efbf8")

var HookAddresses = []common.Address{
	common.HexToAddress("0x0000000aa232009084bd71a5797d089aa4edfad4"),
}
