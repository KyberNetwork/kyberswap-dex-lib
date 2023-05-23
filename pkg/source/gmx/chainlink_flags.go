package gmx

type ChainlinkFlags struct {
	Flags map[string]bool `json:"flags"`
}

const (
	chainlinkFlagsMethodGetFlag = "getFlag"
)
