package polmatic

import _ "embed"

//go:embed abi/PolygonMigration.json
var polygonMigrationABIData []byte

//go:embed abi/ERC20.json
var erc20ABIData []byte
