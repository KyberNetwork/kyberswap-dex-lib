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

	// OpSignatureCacheTTL caps how long GetOpSignatures may serve an operator signature
	// from memory instead of asking the limit-order backend. 0 disables the cache.
	OpSignatureCacheTTL durationjson.Duration `json:"opSignatureCacheTTL"`
	// OpSignatureValidityMargin is how far past now a cached signature must still be valid
	// to be reused; defaults to defaultOpSignatureValidityMargin.
	OpSignatureValidityMargin durationjson.Duration `json:"opSignatureValidityMargin"`
}
