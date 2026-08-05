package tokentax

import (
	"github.com/holiman/uint256"
)

// TaxInfo is the normalized transfer-tax state persisted in pool Extra.
// Tax rates are expressed in basis points.
type TaxInfo struct {
	Token      string       `json:"token,omitempty"`
	BuyTaxBps  *uint256.Int `json:"buyTax,omitempty"`
	SellTaxBps *uint256.Int `json:"sellTax,omitempty"`
	Checked    bool         `json:"checked,omitempty"`
	// TaxCheckVersion marks which tax-detection mechanism last wrote this TaxInfo. A value below
	// the caller's current version means Checked was set by an older/narrower mechanism and must
	// not be trusted as-is - the caller should force one recheck before relying on it again.
	TaxCheckVersion int `json:"taxCheckVersion,omitempty"`
}

// Handler applies normalized transfer tax around the AMM calculation.
// Its zero value is a no-op handler.
type Handler struct {
	TokenAddress string       `msgpack:"tokenAddress"`
	BuyTaxBps    *uint256.Int `msgpack:"buyTaxBps"`
	SellTaxBps   *uint256.Int `msgpack:"sellTaxBps"`
}
