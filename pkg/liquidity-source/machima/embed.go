package machima

import _ "embed"

//go:embed abi/pool.json
var poolABIBytes []byte

//go:embed abi/clank_now.json
var clankNowABIBytes []byte

//go:embed abi/token.json
var tokenABIBytes []byte

//go:embed abi/swap_adapter.json
var swapAdapterABIBytes []byte
