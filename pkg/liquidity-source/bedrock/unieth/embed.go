package unieth

import _ "embed"

//go:embed abis/RockXETH.json
var rockXETHABIJson []byte

//go:embed abis/Staking.json
var stakingABIJson []byte
