package limitorder

import (
	"net/http"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type Config struct {
	DexID             string `json:"dexID"`
	LimitOrderHTTPUrl string `json:"limitOrderHTTPUrl"`
	ChainID           uint   `json:"chainID"`
	SupportMultiSCs   bool   `json:"supportMultiSCs"`

	ContractAddresses []string `json:"contractAddresses"`

	// default=false -> include orders with insufficient balance/allowance
	DisableInsufficientBalance bool `json:"disableInsufficientBalance"`

	HTTPTimeout    durationjson.Duration `json:"httpTimeout"`
	HTTPRetryCount int                   `json:"httpRetryCount"`
	// HTTPClient lets callers supply their own Transport. http.DefaultTransport keeps only
	// 2 idle connections per host, which starves callers that fan RFQ out concurrently.
	HTTPClient *http.Client `json:"-"`
}
