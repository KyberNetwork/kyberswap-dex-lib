package reth

type Gas struct {
	Deposit int64
	Burn    int64
}

var (
	defaultGas = Gas{Deposit: 200000, Burn: 130000}
)
