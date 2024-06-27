package rsethalt1

import _ "embed"

//go:embed abis/rseth_pool.json
var rsETHPool []byte

//go:embed abis/wsteth_eth_oracle.json
var wstethETHOracle []byte
