package eth

import (
	"regexp"

	"github.com/ethereum/go-ethereum/common"
)

var re = regexp.MustCompile("^0x[0-9a-fA-F]{40}$")

var AddressZero common.Address

func ValidateAddress(address string) bool {
	return re.MatchString(address)
}

func IsZeroAddress(address common.Address) bool {
	return address == AddressZero
}
