package types

// ClientData contains data emitted by aggregation router ClientData event
// ref: https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/develop/contracts/MetaAggregationRouterV2.sol#L186
type ClientData struct {
	Source       string
	AmountInUSD  string
	AmountOutUSD string
	Referral     string
	Flags        uint32
}
