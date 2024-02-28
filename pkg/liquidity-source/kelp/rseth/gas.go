package rseth

type Gas struct {
	DepositAsset int64
}

var defaultGas = Gas{
	DepositAsset: 330000,
}
