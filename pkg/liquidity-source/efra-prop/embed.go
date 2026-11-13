package efraprop

import _ "embed"

// PropAmmContract's ABI, trimmed to USDC() and quoteExactIn().

//go:embed abis/EfraPropAMM.json
var abiJSON []byte
