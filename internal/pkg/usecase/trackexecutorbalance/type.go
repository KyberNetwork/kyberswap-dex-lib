package trackexecutor

type SubgraphExecutorExchangesResponse struct {
	ExecutorExchanges []ExchangeEvent `json:"executorExchanges"`
}

type SubgraphRouterSwappedResponse struct {
	SwappedEvents []SwappedEvent `json:"routerSwappeds"`
}

type SubgraphPoolApprovalsResponse struct {
	PoolApprovals []PoolApprovalEvent `json:"executorApprovals"`
}

type ExchangeEvent struct {
	Executor    string `json:"executor"`
	Tx          string `json:"tx"`
	Id          string `json:"id"`
	Pair        string `json:"pair"`
	Token       string `json:"token"`
	BlockNumber string `json:"blockNumber"`

	LogIndex uint32
}

type PoolApprovalEvent struct {
	Executor    string `json:"executor"`
	Token       string `json:"token"`
	Spender     string `json:"spender"`
	BlockNumber string `json:"blockNumber"`
}

type SwappedEvent struct {
	Tx          string `json:"tx"`
	TokenIn     string `json:"tokenIn"`
	TokenOut    string `json:"tokenOut"`
	BlockNumber string `json:"blockNumber"`
}

type SwapInfo struct {
	Pair     string
	TokenIn  string
	TokenOut string
}
