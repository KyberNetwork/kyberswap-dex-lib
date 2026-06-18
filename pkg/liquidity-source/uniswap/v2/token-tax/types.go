package tokentax

import (
	"github.com/KyberNetwork/ethrpc"
	"github.com/holiman/uint256"
)

// TaxInfo is the normalized transfer-tax state persisted in pool Extra.
// Tax rates are expressed in basis points.
type TaxInfo struct {
	Protocol   string       `json:"protocol,omitempty"`
	Token      string       `json:"token,omitempty"`
	BuyTaxBps  *uint256.Int `json:"buyTax,omitempty"`
	SellTaxBps *uint256.Int `json:"sellTax,omitempty"`
	Checked    bool         `json:"checked,omitempty"`
}

// Tracker owns the protocol-specific calls it adds to a shared multicall and resolves their outputs.
type Tracker interface {
	AddCalls(*ethrpc.Request)
	Resolve(*ethrpc.Response) TaxInfo
}

// Handler applies normalized transfer tax around the AMM calculation.
// Its zero value is a no-op handler.
type Handler struct {
	TokenAddress string       `msgpack:"tokenAddress"`
	BuyTaxBps    *uint256.Int `msgpack:"buyTaxBps"`
	SellTaxBps   *uint256.Int `msgpack:"sellTaxBps"`
}
