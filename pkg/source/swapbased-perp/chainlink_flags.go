//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple ChainlinkFlags

package swapbasedperp

type ChainlinkFlags struct {
	Flags map[string]bool `json:"flags"`
}

const (
	chainlinkFlagsMethodGetFlag = "getFlag"
)

func (f *ChainlinkFlags) GetFlag(address string) bool {
	return f.Flags[address]
}
