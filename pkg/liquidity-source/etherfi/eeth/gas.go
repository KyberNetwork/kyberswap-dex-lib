//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple Gas

package eeth

type Gas struct {
	Deposit int64
}

var (
	defaultGas = Gas{Deposit: 70000}
)
