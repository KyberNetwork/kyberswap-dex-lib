package aevmclient

type Config struct {
	ServerURLs          []string `json:"serverUrls"`
	PublishingPoolsURLs []string `json:"publishingPoolsUrls"`

	RetryOnTimeoutMs          uint64 `json:"retryOnTimeOutMs"`
	FindrouteRetryOnTimeoutMs uint64 `json:"findRouteRetryOnTimeOutMs"`
	MaxRetry                  uint64 `json:"maxRetry"`
}
