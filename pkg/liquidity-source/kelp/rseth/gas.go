//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple Gas

package rseth

type Gas struct {
	DepositAsset int64
}

var defaultGas = Gas{
	DepositAsset: 330000,
}
