package netstaking

import _ "embed"

//go:embed abi/Staking.json
var stakingABIJson []byte

//go:embed abi/StakedNET.json
var stakedNETABIJson []byte
