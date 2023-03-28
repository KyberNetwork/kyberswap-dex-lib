package eth

import (
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

var re = regexp.MustCompile("^0x[0-9a-fA-F]{40}$")

func ValidateAddress(address string) bool {
	return re.MatchString(address)
}

func IsZeroAddress(address common.Address) bool {
	return strings.EqualFold(address.String(), constant.AddressZero)
}
