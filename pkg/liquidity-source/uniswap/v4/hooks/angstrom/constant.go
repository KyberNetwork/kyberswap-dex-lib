package angstrom

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const (
	PathGetAttestations = "/getAttestations"
)

var (
	ONE_E6 = big.NewInt(1_000_000)

	Adapter = common.HexToAddress("0xb535aeb27335b91e1b5bccbd64888ba7574efbf8")

	HookAddresses = []common.Address{common.HexToAddress("0x0000000aa232009084Bd71A5797d089AA4Edfad4")}

	L2HookAddresses []common.Address
)
