package liquid

import _ "embed"

//go:embed abis/Accountant.json
var accountantABIJson []byte

//go:embed abis/Teller.json
var tellerABIJson []byte
