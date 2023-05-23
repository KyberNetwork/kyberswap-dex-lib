package elastic

import _ "embed"

//go:embed abis/ElasticPool.json
var elasticPoolJson []byte

//go:embed abis/ERC20.json
var erc20Json []byte
