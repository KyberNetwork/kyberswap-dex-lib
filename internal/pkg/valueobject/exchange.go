package valueobject

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Exchange = valueobject.Exchange

func IsAnExchange(exchange Exchange) bool {
	return valueobject.IsAMMSource(exchange) || valueobject.IsRFQSource(exchange)
}
