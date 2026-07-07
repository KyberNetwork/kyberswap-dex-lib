package gmx

type ChainlinkFlags struct {
	Flags map[string]bool `json:"flags,omitempty"`
}

const (
	chainlinkFlagsMethodGetFlag = "getFlag"
)

func (f *ChainlinkFlags) GetFlag(address string) bool {
	return f.Flags[address]
}
