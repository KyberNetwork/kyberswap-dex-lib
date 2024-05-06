//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple Gas

package weeth

type Gas struct {
	Wrap   int64
	Unwrap int64
}

var (
	defaultGas = Gas{Wrap: 140000, Unwrap: 80000}
)
