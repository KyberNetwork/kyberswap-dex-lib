package gmxglp

type ChainlinkFlags struct {
	Flags map[string]bool `json:"flags"`
}

const (
	chainlinkFlagsMethodGetFlag = "getFlag"
)

func (f *ChainlinkFlags) GetFlag(address string) bool {
	return f.Flags[address]
}
