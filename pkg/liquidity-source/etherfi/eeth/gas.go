package eeth

type Gas struct {
	Deposit int64
}

var (
	defaultGas = Gas{Deposit: 70000}
)
