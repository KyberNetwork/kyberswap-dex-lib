package eth

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func IsZeroAddress(address common.Address) bool {
	return strings.EqualFold(address.String(), valueobject.ZeroAddress)
}
