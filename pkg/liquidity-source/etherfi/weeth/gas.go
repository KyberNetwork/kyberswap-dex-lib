package weeth

type Gas struct {
	Wrap   int64
	Unwrap int64
}

var (
	defaultGas = Gas{Wrap: 140000, Unwrap: 80000}
)
