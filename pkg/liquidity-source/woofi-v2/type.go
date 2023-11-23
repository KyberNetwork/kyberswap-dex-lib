package woofiv2

import "github.com/holiman/uint256"

// DecimalInfo
// https://github.com/woonetwork/WooPoolV2/blob/e4fc06d357e5f14421c798bf57a251f865b26578/contracts/WooPPV2.sol#L58
type DecimalInfo struct {
	priceDec *uint256.Int // 10**(price_decimal)
	quoteDec *uint256.Int // 10**(quote_decimal)
	baseDec  *uint256.Int // 10**(base_decimal)
}
