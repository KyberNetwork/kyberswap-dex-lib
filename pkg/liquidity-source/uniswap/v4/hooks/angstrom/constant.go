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

	L2HookAddresses = []common.Address{
		common.HexToAddress("0xCD256a2f4574CB6acA4837313ad225d2fe1De5Cf"),
		common.HexToAddress("0x7Fa49D29481b6D168505Ccde26635e204c09e5CF"),
	}
)
