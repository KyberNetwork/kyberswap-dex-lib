package bancorv3

import _ "embed"

//go:embed abis/BancorNetwork.json
var bancorNetworkJSON []byte

//go:embed abis/PoolCollection.json
var poolCollectionJSON []byte
