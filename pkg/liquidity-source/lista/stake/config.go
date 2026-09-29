package stake

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

// PoolConfig identifies one ListaStakeManager (or equivalent) instance. Each instance is
// permanently fixed to one pair (deposit token <-> share token) at deploy time, so a new chain or
// a new deployment just adds an entry here -- no dex-lib code change or version bump needed.
type PoolConfig struct {
	PoolAddress string `json:"poolAddress"`
	// UnderlyingToken is the deposit-side token address. Set it to valueobject.NativeAddress
	// (0xEeee...) when the contract's deposit is native/payable (e.g. Lista's own
	// deposit() external payable) -- the pool's graph-space token is still stored wrapped (e.g.
	// WBNB), but the pool remembers it was configured native so SwapReceiveNativeIn/
	// SwapReturnNativeOut report it truthfully. Set it to the plain wrapped/ERC20 address instead
	// for a variant whose deposit takes the token via transferFrom, no native handling needed.
	UnderlyingToken string `json:"underlyingToken"`
	// ShareToken is the liquid-staking-derivative token minted on deposit (e.g. slisBNB).
	ShareToken string `json:"shareToken"`
}

type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainID"`
	Pools   []PoolConfig        `json:"pools"`
}
