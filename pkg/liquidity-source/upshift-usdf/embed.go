package upshiftusdf

import _ "embed"

//go:embed abi/Vault.json
var vaultABIJson []byte

//go:embed abi/ReceiptToken.json
var receiptTokenABIJson []byte
