package clientdata

import (
	"errors"
	"reflect"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"
)

const (
	KeySetTokenInMarketPriceAvailable  uint32 = 1 << iota // 1
	KeySetTokenOutMarketPriceAvailable                    // 2
)

const (
	MaximumSupportedFlags = 32
)

var (
	ErrTooManyFlags = errors.New("too many flags (max 32)")
)

// AddFlag adds a flag to the client data flags
func AddFlag(beforeFlag uint32, flag uint32) (afterFlag uint32) {
	afterFlag = beforeFlag | flag

	return
}

// ConvertFlagsToBitInteger create a flags to be stored inside client data.
// The maximum number of supported flags is 32
// Each bit (0 or 1) in the result represents a flag
// so that we must keep them in a specific order
// and that specific order is maintained here
func ConvertFlagsToBitInteger(
	flags valueobject.Flags,
) (uint32, error) {
	var result uint32
	// get the number of flags
	numFlags := reflect.TypeOf(flags).NumField()
	if numFlags > MaximumSupportedFlags {
		return 0, ErrTooManyFlags
	}

	// The 1st bit from the right is TokenInMarketPriceAvailable
	if flags.TokenInMarketPriceAvailable {
		result = AddFlag(result, KeySetTokenInMarketPriceAvailable)
	}

	// The 2nd bit from the right is TokenOutMarketPriceAvailable
	if flags.TokenOutMarketPriceAvailable {
		result = AddFlag(result, KeySetTokenOutMarketPriceAvailable)
	}

	return result, nil
}
