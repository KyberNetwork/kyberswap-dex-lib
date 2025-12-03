package angstrom

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const (
	Handler               = "uniswap-v4-angstrom"
	GET_ATTESTATIONS_PATH = "/getAttestations"
)

var (
	ONE_E6 = big.NewInt(1_000_000)

	Adapter = common.HexToAddress("0xb535aeb27335b91e1b5bccbd64888ba7574efbf8")

	HookAddresses = []common.Address{common.HexToAddress("0x0000000aa232009084bd71a5797d089aa4edfad4")}
)
