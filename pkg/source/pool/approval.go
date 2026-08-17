package pool

type ApprovalInfo struct {
	ApprovalAddress string `json:"approvalAddress"`
}

type MetaInfo struct {
	ApprovalAddress string `json:"approvalAddress"`
	BlockNumber     uint64 `json:"blockNumber"`
}
