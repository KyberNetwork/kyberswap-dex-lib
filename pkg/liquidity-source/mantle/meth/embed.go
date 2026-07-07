package meth

import _ "embed"

//go:embed abis/MantleLSPStaking.json
var stakingABIJSON []byte

//go:embed abis/MantlePauser.json
var pauserABIJSON []byte

//go:embed abis/METH.json
var methABIJSON []byte
