package composablestable

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"

type PairTypes int

const (
	BptToToken PairTypes = iota
	TokenToBpt
	TokenToToken
)

var (
	AmpPrecision = bignumber.TenPowInt(3)
)
