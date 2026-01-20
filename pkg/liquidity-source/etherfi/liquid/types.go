package liquid

type Extra struct {
	SupportedWithdraw []bool `json:"supportedWithdraw"`
}

type Gas struct {
	Deposit  int64
	Withdraw int64
}
