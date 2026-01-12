package ekubov3

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const DexType = valueobject.ExchangeEkuboV3

var (
	ErrZeroAmount = errors.New("zero amount")
	ErrReorg      = errors.New("reorg detected")
)
