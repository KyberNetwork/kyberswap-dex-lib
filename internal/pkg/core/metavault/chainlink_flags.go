package metavault

type ChainlinkFlags struct {
	Flags map[string]bool `json:"flags"`
}

func (f *ChainlinkFlags) GetFlag(address string) bool {
	return f.Flags[address]
}
