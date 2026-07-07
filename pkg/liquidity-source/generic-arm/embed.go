package genericarm

import _ "embed"

//go:embed abis/lidoarm.json
var lidoArmABIData []byte

//go:embed abis/ERC4626.json
var ERC626ABIData []byte
