package eth

import (
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

var re = regexp.MustCompile("^0x[0-9a-fA-F]{40}$")

const AddressZero = "0x0000000000000000000000000000000000000000"

func ValidateAddress(address string) bool {
	return re.MatchString(address)
}

func IsZeroAddress(address common.Address) bool {
	return strings.EqualFold(address.String(), AddressZero)
}
